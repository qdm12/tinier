package metadata

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainedBoxes(t *testing.T) {
	t.Run("two boxes", func(t *testing.T) {
		contents := append(boxBytes("one_", []byte("first")), boxBytes("two_", nil)...)

		boxes, err := containedBoxes(contents)
		require.NoError(t, err)
		require.Len(t, boxes, 2)
		assert.Equal(t, box{name: "one_", contents: []byte("first")}, boxes[0])
		assert.Equal(t, box{name: "two_", contents: []byte{}}, boxes[1])
	})

	t.Run("no contents", func(t *testing.T) {
		boxes, err := containedBoxes(nil)
		require.NoError(t, err)
		assert.Empty(t, boxes)
	})

	// A file whose boxes announce more bytes than it holds is what a truncated
	// download is made of.
	for _, name := range []string{"a header cut short", "a box longer than the contents"} {
		t.Run(name, func(t *testing.T) {
			var contents []byte
			switch name {
			case "a header cut short":
				contents = boxBytes("one_", []byte("first"))[:6]
			default:
				contents = append(boxHeader("one_", 1000), []byte("ten bytes")...)
			}

			_, err := containedBoxes(contents)
			assert.ErrorIs(t, err, ErrInvalidImage)
		})
	}

	t.Run("a box of length zero reaches the end of the contents", func(t *testing.T) {
		// A box of a file on its own announces a length of zero, which the code
		// reads as the box reaching the end of what holds it.
		contents := append([]byte{0, 0, 0, 0}, append([]byte("rest"), []byte("left over")...)...)

		boxes, err := containedBoxes(contents)
		require.NoError(t, err)
		require.Len(t, boxes, 1)
		assert.Equal(t, "rest", boxes[0].name)
		assert.Equal(t, "left over", string(boxes[0].contents))
	})

	t.Run("a box of length one holds its length next", func(t *testing.T) {
		contents := []byte{0, 0, 0, 1}
		contents = append(contents, "exte"...)
		contents = binary.BigEndian.AppendUint64(contents, uint64(boxLengthSize+boxNameSize+longBoxLengthSize+7))
		contents = append(contents, []byte("payload")...)

		boxes, err := containedBoxes(contents)
		require.NoError(t, err)
		require.Len(t, boxes, 1)
		assert.Equal(t, "exte", boxes[0].name)
		assert.Equal(t, "payload", string(boxes[0].contents))
	})

	t.Run("a box shorter than its header", func(t *testing.T) {
		_, err := containedBoxes(append(boxHeader("one_", 6), []byte("abcdef")...))
		assert.ErrorIs(t, err, ErrInvalidImage)
	})
}

func TestReadTopLevelBox(t *testing.T) {
	first := boxBytes("ftyp", []byte("avif"))
	second := boxBytes("mdat", []byte("the pixels of the file"))
	file := make([]byte, 0, len(first)+len(second))
	file = append(file, first...)
	file = append(file, second...)
	fileSize := int64(len(file))

	t.Run("the first box of a file", func(t *testing.T) {
		h, err := readTopLevelBox(bytes.NewReader(file), 0, fileSize)
		require.NoError(t, err)
		assert.Equal(t, "ftyp", h.name)
		assert.Equal(t, int64(boxLengthSize+boxNameSize), h.offset)
		assert.Equal(t, int64(len("avif")), h.length)
		assert.Equal(t, int64(len(first)), h.next)
	})

	t.Run("the box which follows", func(t *testing.T) {
		h, err := readTopLevelBox(bytes.NewReader(file), int64(len(first)), fileSize)
		require.NoError(t, err)
		assert.Equal(t, "mdat", h.name)
		assert.Equal(t, int64(len(first)+boxLengthSize+boxNameSize), h.offset)
		assert.Equal(t, int64(len("the pixels of the file")), h.length)
		assert.Equal(t, fileSize, h.next)
	})

	t.Run("past the last box of the file", func(t *testing.T) {
		_, err := readTopLevelBox(bytes.NewReader(file), fileSize, fileSize)
		assert.ErrorIs(t, err, ErrInvalidImage)
	})

	t.Run("a box of length zero reaches the end of the file", func(t *testing.T) {
		data := append(boxHeader("mdat", 0), []byte("the rest of the file")...)

		h, err := readTopLevelBox(bytes.NewReader(data), 0, int64(len(data)))
		require.NoError(t, err)
		assert.Equal(t, "mdat", h.name)
		assert.Equal(t, int64(len("the rest of the file")), h.length)
		assert.Equal(t, int64(len(data)), h.next)
	})

	t.Run("a box of length one holds its length next", func(t *testing.T) {
		data := []byte{0, 0, 0, 1}
		data = append(data, "mdat"...)
		data = binary.BigEndian.AppendUint64(data, uint64(boxLengthSize+boxNameSize+longBoxLengthSize+3))
		data = append(data, []byte("png")...)

		h, err := readTopLevelBox(bytes.NewReader(data), 0, int64(len(data)))
		require.NoError(t, err)
		assert.Equal(t, "mdat", h.name)
		assert.Equal(t, int64(boxLengthSize+boxNameSize+longBoxLengthSize), h.offset)
		assert.Equal(t, int64(3), h.length)
	})

	t.Run("a box which ends past the file", func(t *testing.T) {
		data := append(boxHeader("mdat", 1000), []byte("ten bytes")...)

		_, err := readTopLevelBox(bytes.NewReader(data), 0, int64(len(data)))
		assert.ErrorIs(t, err, ErrInvalidImage)
	})

	t.Run("a box shorter than its header", func(t *testing.T) {
		data := append(boxHeader("mdat", 4), []byte("mdat")...)

		_, err := readTopLevelBox(bytes.NewReader(data), 0, int64(len(data)))
		assert.ErrorIs(t, err, ErrInvalidImage)
	})
}

func TestTopLevelBoxContents(t *testing.T) {
	file := append(boxBytes("ftyp", []byte("avif")), boxBytes("meta", []byte("the metadata"))...)

	t.Run("the box of the metadata of a file", func(t *testing.T) {
		contents, found, err := topLevelBoxContents(bytes.NewReader(file), int64(len(file)), metaBoxName)
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, "the metadata", string(contents))
	})

	t.Run("a box the file does not hold", func(t *testing.T) {
		contents, found, err := topLevelBoxContents(bytes.NewReader(file), int64(len(file)), "iinf")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, contents)
	})

	t.Run("a box announcing more bytes than a file of metadata can hold", func(t *testing.T) {
		data := append(boxHeader(metaBoxName, maxMetadataBoxLength+1), make([]byte, 8)...)

		_, _, err := topLevelBoxContents(bytes.NewReader(data), int64(len(data)), metaBoxName)
		assert.ErrorIs(t, err, ErrInvalidImage)
	})
}

func TestSigned(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		want  int64
		ok    bool
	}{
		{"zero", 0, 0, true},
		{"the largest signed number", math.MaxInt64, math.MaxInt64, true},
		{"above the largest signed number", math.MaxUint64, 0, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, ok := signed(test.value)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.want, value)
		})
	}
}

func TestCursor(t *testing.T) {
	contents := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 'n', 'a', 'm', 'e'}

	t.Run("the integers of the sizes of the format", func(t *testing.T) {
		c := &cursor{contents: contents}
		assert.Equal(t, uint32(1), c.uint(1))
		assert.Equal(t, uint32(0x0203), c.uint(2))
		assert.Equal(t, uint32(0x04050607), c.uint(4))
		assert.Equal(t, uint32(0), c.uint(0), "a field of no byte is not read")
		require.NoError(t, c.err)
		assert.Equal(t, 7, c.position)
		c.skip(1)
		require.NoError(t, c.err)
		assert.Equal(t, "name", c.name())
		assert.Equal(t, len(contents), c.position)
	})

	t.Run("a long integer", func(t *testing.T) {
		c := &cursor{contents: contents}
		assert.Equal(t, int64(0x0102030405060708), c.long(8))
		require.NoError(t, c.err)
	})

	t.Run("a read past the contents fails the cursor", func(t *testing.T) {
		c := &cursor{contents: []byte{1, 2}}
		assert.Equal(t, uint32(0x0102), c.uint(2))
		assert.Equal(t, uint32(0), c.uint(4))
		assert.ErrorIs(t, c.err, ErrInvalidImage)
		// Once failed, the cursor never reads anything again.
		c.skip(1)
		assert.ErrorIs(t, c.err, ErrInvalidImage)
		assert.Equal(t, "", c.name())
	})

	t.Run("a negative size fails the cursor", func(t *testing.T) {
		c := &cursor{contents: contents}
		assert.Equal(t, int64(0), c.long(-1))
		assert.ErrorIs(t, c.err, ErrInvalidImage)
	})

	t.Run("a name cut short fails the cursor", func(t *testing.T) {
		c := &cursor{contents: []byte("nam")}
		assert.Equal(t, "", c.name())
		assert.ErrorIs(t, c.err, ErrInvalidImage)
	})
}

// boxBytes returns the bytes of a box of a base media file, of its header and of
// its contents.
func boxBytes(name string, contents []byte) []byte {
	// A fixture of a few bytes, of the size a box of a file of metadata can be.
	size := uint32(boxLengthSize + boxNameSize + len(contents)) //nolint:gosec
	return append(boxHeader(name, size), contents...)
}

// boxHeader returns the header of a box of a base media file, announcing the
// given length: the four bytes of the length and the four of the name.
func boxHeader(name string, length uint32) []byte {
	header := binary.BigEndian.AppendUint32(nil, length)

	return append(header, name...)
}
