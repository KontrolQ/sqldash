package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

type LogLevel string

const (
	LevelDebug   LogLevel = "debug"
	LevelInfo    LogLevel = "info"
	LevelWarn    LogLevel = "warn"
	LevelError   LogLevel = "error"
	LevelSuccess LogLevel = "success"
)

func formatLevel(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		encoder.AppendString(LevelColorDebug)
	case zapcore.WarnLevel:
		encoder.AppendString(LevelColorWarn)
	case zapcore.ErrorLevel:
		encoder.AppendString(LevelColorError)
	default:
		encoder.AppendString(LevelColorInfo)
	}
}

func formatPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	padding := ""
	if len(prefix) < PrefixWidth {
		padding = strings.Repeat(" ", PrefixWidth-len(prefix))
	}

	return PrefixColor + "[" + prefix + "]" + AnsiReset + padding
}

func colorizeMessage(level LogLevel, message string) string {
	switch level {
	case LevelDebug:
		return MessageColorDebug + message + AnsiReset
	case LevelWarn:
		return MessageColorWarn + message + AnsiReset
	case LevelError:
		return MessageColorError + message + AnsiReset
	case LevelSuccess:
		return MessageColorSuccess + message + AnsiReset
	default:
		return MessageColorInfo + message + AnsiReset
	}
}

func buildFullMessage(level LogLevel, prefix string, message any) string {
	return formatPrefix(prefix) + colorizeMessage(level, fmt.Sprint(message))
}
