ARG ALPINE_VERSION=3.16
ARG GO_VERSION=1.19
ARG XCPUTRANSLATE_VERSION=v0.6.0
ARG GOLANGCI_LINT_VERSION=v1.50.1
ARG MOCKGEN_VERSION=v1.6.0

FROM --platform=${BUILDPLATFORM} qmcgaw/xcputranslate:${XCPUTRANSLATE_VERSION} AS xcputranslate
FROM --platform=${BUILDPLATFORM} qmcgaw/binpot:golangci-lint-${GOLANGCI_LINT_VERSION} AS golangci-lint
FROM --platform=${BUILDPLATFORM} qmcgaw/binpot:mockgen-${MOCKGEN_VERSION} AS mockgen

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
ENV CGO_ENABLED=0
RUN apk --update add git g++ findutils
WORKDIR /tmp/gobuild
COPY --from=xcputranslate /xcputranslate /usr/local/bin/xcputranslate
COPY --from=golangci-lint /bin /go/bin/golangci-lint
COPY --from=mockgen /bin /go/bin/mockgen
# Copy repository code and install Go dependencies
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/

FROM --platform=${BUILDPLATFORM} base AS test
ENV CGO_ENABLED=1

FROM --platform=${BUILDPLATFORM} base AS lint
COPY .golangci.yml ./
RUN golangci-lint run --timeout=10m

FROM --platform=${BUILDPLATFORM} base AS mocks
RUN git init && \
  git config user.email ci@localhost && \
  git config user.name ci && \
  git config core.fileMode false && \
  git add -A && \
  git commit -m "snapshot" && \
  grep -lr -E '^// Code generated by MockGen\. DO NOT EDIT\.$' . | xargs -r -d '\n' rm && \
  go generate -run "mockgen" ./... && \
  git diff --exit-code && \
  rm -rf .git/

FROM --platform=${BUILDPLATFORM} base AS build
ARG TARGETPLATFORM
ARG VERSION=unknown
ARG CREATED="an unknown date"
ARG COMMIT=unknown
RUN GOARCH="$(xcputranslate translate -targetplatform ${TARGETPLATFORM} -field arch)" \
  GOARM="$(xcputranslate translate -targetplatform ${TARGETPLATFORM} -field arm)" \
  go build -trimpath -ldflags="-s -w \
  -X 'main.version=$VERSION' \
  -X 'main.commit=$COMMIT' \
  -X 'main.buildDate=$CREATED' \
  " -o entrypoint cmd/tinier/main.go

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ffmpeg
ENTRYPOINT ["/tinier"]
USER 1000
ENV \
  TINIER_INPUT_DIR_PATH=/input \
  TINIER_OUTPUT_DIR_PATH=/output \
  TINIER_FFMPEG_PATH= \
  TINIER_FFMPEG_MIN_VERSION=5.0.1 \
  TINIER_OVERRIDE_OUTPUT=off \
  TINIER_VIDEO_SCALE="1280:-1" \
  TINIER_VIDEO_PRESET=8 \
  TINIER_VIDEO_CODEC=libsvtav1 \
  TINIER_VIDEO_OUTPUT_EXTENSION=".mp4" \
  TINIER_VIDEO_EXTENSIONS=".mp4,.mov,.avi" \
  TINIER_VIDEO_SKIP=no \
  TINIER_VIDEO_CRF=23 \
  TINIER_IMAGE_SCALE=5 \
  TINIER_IMAGE_OUTPUT_EXTENSION=".jpg" \
  TINIER_IMAGE_EXTENSIONS=".jpg,.jpeg,.png" \
  TINIER_IMAGE_SKIP=no \
  TINIER_IMAGE_QSCALE=5 \
  TINIER_AUDIO_CODEC=libmp3lame \
  TINIER_AUDIO_OUTPUT_EXTENSION=".mp3" \
  TINIER_AUDIO_EXTENSIONS=".mp3,.flac" \
  TINIER_AUDIO_SKIP=no \
  TINIER_AUDIO_QSCALE=5
ARG VERSION=unknown
ARG CREATED="an unknown date"
ARG COMMIT=unknown
LABEL \
  org.opencontainers.image.authors="quentin.mcgaw@gmail.com" \
  org.opencontainers.image.created=$CREATED \
  org.opencontainers.image.version=$VERSION \
  org.opencontainers.image.revision=$COMMIT \
  org.opencontainers.image.url="https://github.com/qdm12/tinier" \
  org.opencontainers.image.documentation="https://github.com/qdm12/tinier" \
  org.opencontainers.image.source="https://github.com/qdm12/tinier" \
  org.opencontainers.image.title="tinier" \
  org.opencontainers.image.description=""
COPY --from=build --chown=1000 /tmp/gobuild/entrypoint /tinier
