package zaploki

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogHandler interface {
	Sync() error
	Proceed(entry zapcore.Entry, fields []zapcore.Field) error
}

type Core struct {
	zapcore.LevelEnabler
	logHandler LogHandler
	fields     map[string]zapcore.Field
}

var (
	_ zapcore.Core = (*Core)(nil)
)

func NewCore(lc LogHandler, lv zapcore.LevelEnabler) zapcore.Core {
	return &Core{
		LevelEnabler: lv,
		logHandler:   lc,
	}
}

func NewCoreWithCreateLogger(lc LogHandler, zapConfig zap.Config) (*zap.Logger, error) {
	return zapConfig.Build(
		zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewTee(c, NewCore(lc, zapConfig.Level))
		}),
		// zap.AddCallerSkip(1),
	)
}

func (loki *Core) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if loki.Enabled(entry.Level) {
		return ce.AddCore(entry, loki)
	}
	return ce
}

func (loki *Core) Sync() error {
	return loki.logHandler.Sync()
}

func (loki *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := loki.clone()
	clone.addFields(fields)
	return clone
}

func (loki *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	err := loki.logHandler.Proceed(entry, fields)
	if err != nil {
		return err
	}
	if entry.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output.
		// Ignore Sync errors, pending a clean solution to issue #370.
		_ = loki.Sync()
	}
	return nil
}

func (loki *Core) clone() *Core {
	return &Core{
		LevelEnabler: loki.LevelEnabler,
		logHandler:   loki.logHandler,
	}
}

func (loki *Core) addFields(fields []zapcore.Field) {
	for _, f := range fields {
		loki.fields[f.Key] = f
	}
}
