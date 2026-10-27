package logger

import (
	"regexp"
	"strings"

	"go.uber.org/zap/zapcore"
)

type OutputFormat string

const (
	PlainOutput    OutputFormat = "plain"
	PostgresOutput OutputFormat = "postgres"
	LibSQLOutput   OutputFormat = "libsql"
)

var (
	outputANSI   = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	postgresLine = regexp.MustCompile(`^(?:[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9:.]+ [A-Za-z0-9:+-]+ )?(\[[0-9]+\] )?(DEBUG[1-5]?|INFO|NOTICE|WARNING|ERROR|LOG|FATAL|PANIC|DETAIL|HINT|CONTEXT|STATEMENT):\s?(.*)$`)
	libsqlLine   = regexp.MustCompile(`^(?:[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+(?:Z|[+-][0-9:]+)\s+)?(TRACE|DEBUG|INFO|WARN|ERROR)\s+(.*)$`)
)

func processLine(format OutputFormat, line string, previous zapcore.Level) (zapcore.Level, string) {
	line = outputANSI.ReplaceAllString(strings.TrimSuffix(line, "\r"), "")
	if format == PostgresOutput {
		if parts := postgresLine.FindStringSubmatch(line); parts != nil {
			if parts[2] == "DETAIL" || parts[2] == "HINT" || parts[2] == "CONTEXT" || parts[2] == "STATEMENT" {
				return previous, parts[1] + parts[2] + ": " + parts[3]
			}
			return processLevel(parts[2]), parts[1] + parts[3]
		}
	} else if format == LibSQLOutput {
		if parts := libsqlLine.FindStringSubmatch(line); parts != nil {
			return processLevel(parts[1]), parts[2]
		}
	}
	return previous, line
}

func processLevel(level string) zapcore.Level {
	switch level {
	case "TRACE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3", "DEBUG4", "DEBUG5":
		return zapcore.DebugLevel
	case "WARN", "WARNING":
		return zapcore.WarnLevel
	case "ERROR", "FATAL", "PANIC":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func processLabel(level zapcore.Level) LogLevel {
	switch level {
	case zapcore.DebugLevel:
		return LevelDebug
	case zapcore.WarnLevel:
		return LevelWarn
	case zapcore.ErrorLevel:
		return LevelError
	default:
		return LevelInfo
	}
}
