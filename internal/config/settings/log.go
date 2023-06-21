package settings

import (
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gotree"
	"github.com/qdm12/log"
)

type Log struct {
	Level log.Level
}

func (l *Log) setDefaults() {
	l.Level = gosettings.DefaultNumber(l.Level, log.LevelInfo)
}

func (l *Log) validate() (err error) {
	return nil
}

func (l *Log) mergeWith(other Log) {
	l.Level = gosettings.MergeWithNumber(l.Level, other.Level)
}

func (l *Log) overrideWith(other Log) {
	l.Level = gosettings.OverrideWithNumber(l.Level, other.Level)
}

func (l *Log) toLinesNode() (node *gotree.Node) {
	node = gotree.New("Log")
	node.Appendf("Level %s", l.Level.String())
	return node
}
