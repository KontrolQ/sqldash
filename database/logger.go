package database

import (
	"context"
	"fmt"
	"time"

	"sqldash/config"
	"sqldash/utils/logger"

	gormlogger "gorm.io/gorm/logger"
)

type queryLogger struct {
	level gormlogger.LogLevel
}

func resolveGORMLogLevel() gormlogger.Interface {
	if config.Server.Debug {
		return queryLogger{level: gormlogger.Info}
	}
	return queryLogger{level: gormlogger.Silent}
}

func (self queryLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	self.level = level
	return self
}

func (self queryLogger) Info(ctx context.Context, message string, arguments ...any) {
	if self.level >= gormlogger.Info {
		logger.Infof(LogPrefix, message, arguments...)
	}
}

func (self queryLogger) Warn(ctx context.Context, message string, arguments ...any) {
	if self.level >= gormlogger.Warn {
		logger.Warnf(LogPrefix, message, arguments...)
	}
}

func (self queryLogger) Error(ctx context.Context, message string, arguments ...any) {
	if self.level >= gormlogger.Error {
		logger.Errorf(LogPrefix, message, arguments...)
	}
}

func (self queryLogger) Trace(ctx context.Context, started time.Time, query func() (string, int64), err error) {
	if self.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(started)
	logMessage := logger.Infof
	qualification := ""
	switch {
	case err != nil && self.level >= gormlogger.Error:
		logMessage = logger.Errorf
	case elapsed > SlowQueryThreshold && self.level >= gormlogger.Warn:
		logMessage = logger.Warnf
		qualification = "Slow query "
	case self.level >= gormlogger.Info:
	default:
		return
	}
	statement, rows := query()
	message := fmt.Sprintf("%s[%.3fms] [rows:%d] %s", qualification, float64(elapsed)/float64(time.Millisecond), rows, statement)
	if err != nil {
		message += ": " + err.Error()
	}
	logMessage(LogPrefix, "%s", message)
}
