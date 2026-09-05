package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance    *zap.Logger
	atomicLevel zap.AtomicLevel
)

func init() {
	atomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	encoderConfig := zapcore.EncoderConfig{
		LevelKey:    "level",
		MessageKey:  "msg",
		LineEnding:  "\n",
		EncodeLevel: formatLevel,
	}

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	stdoutSink := zapcore.AddSync(os.Stdout)
	stderrSink := zapcore.AddSync(os.Stderr)

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stdoutSink, zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level < zapcore.WarnLevel && atomicLevel.Enabled(level)
		})),
		zapcore.NewCore(encoder, stderrSink, zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zapcore.WarnLevel && atomicLevel.Enabled(level)
		})),
	)

	instance = zap.New(core, zap.AddCaller())
}

func SetDebug(enabled bool) {
	if enabled {
		atomicLevel.SetLevel(zapcore.DebugLevel)
	} else {
		atomicLevel.SetLevel(zapcore.InfoLevel)
	}
}

func Debugf(prefix string, format string, arguments ...any) {
	emit(LevelDebug, zapcore.DebugLevel, prefix, fmt.Sprintf(format, arguments...))
}

func Infof(prefix string, format string, arguments ...any) {
	emit(LevelInfo, zapcore.InfoLevel, prefix, fmt.Sprintf(format, arguments...))
}

func Successf(prefix string, format string, arguments ...any) {
	emit(LevelSuccess, zapcore.InfoLevel, prefix, fmt.Sprintf(format, arguments...))
}

func Warnf(prefix string, format string, arguments ...any) {
	emit(LevelWarn, zapcore.WarnLevel, prefix, fmt.Sprintf(format, arguments...))
}

func Errorf(prefix string, format string, arguments ...any) {
	emit(LevelError, zapcore.ErrorLevel, prefix, fmt.Sprintf(format, arguments...))
}

func Fatalf(prefix string, format string, arguments ...any) {
	emit(LevelError, zapcore.ErrorLevel, prefix, fmt.Sprintf(format, arguments...))
	os.Exit(1)
}

func emit(levelLabel LogLevel, zapLevel zapcore.Level, prefix string, message any) {
	if instance == nil {
		panic(NotInitialized)
	}

	instance.Log(zapLevel, buildFullMessage(levelLabel, prefix, message))
}
