package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logLevels = map[string]zapcore.Level{
		"debug": zap.DebugLevel,
		"info":  zap.InfoLevel,
		"warn":  zap.WarnLevel,
		"error": zap.ErrorLevel,
		"fatal": zap.FatalLevel,
	}
)

// New ...
func New(lv string) (l *zap.Logger, err error) {
	config := zap.NewProductionConfig()
	config.Encoding = "json"
	config.Level = zap.NewAtomicLevelAt(logLevel(lv))
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	l, err = config.Build()
	if err != nil {
		return
	}
	l = l.WithOptions(zap.AddCallerSkip(1))
	return
}

func logLevel(lv string) zapcore.Level {
	if lv == "" {
		return zap.InfoLevel
	}
	l, ok := logLevels[lv]
	if !ok {
		return zap.InfoLevel
	}
	return l
}
