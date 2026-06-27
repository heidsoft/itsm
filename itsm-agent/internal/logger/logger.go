package logger

import (
	"log"
	"os"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	logger     *log.Logger
	level      = INFO
	prefix     = ""
	output     = os.Stdout
	timeFormat = "2006-01-02 15:04:05"
)

func Init(lvl string, p string) {
	switch lvl {
	case "debug":
		level = DEBUG
	case "info":
		level = INFO
	case "warn":
		level = WARN
	case "error":
		level = ERROR
	}
	prefix = p
	logger = log.New(output, "", 0)
}

func SetLevel(lvl string) {
	switch lvl {
	case "debug":
		level = DEBUG
	case "info":
		level = INFO
	case "warn":
		level = WARN
	case "error":
		level = ERROR
	}
}

func Debug(format string, v ...interface{}) {
	if level <= DEBUG {
		logger.Printf(formatTime()+prefix+" [DEBUG] "+format, v...)
	}
}

func Info(format string, v ...interface{}) {
	if level <= INFO {
		logger.Printf(formatTime()+prefix+" [INFO] "+format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	if level <= WARN {
		logger.Printf(formatTime()+prefix+" [WARN] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if level <= ERROR {
		logger.Printf(formatTime()+prefix+" [ERROR] "+format, v...)
	}
}

func Fatal(format string, v ...interface{}) {
	logger.Printf(formatTime()+prefix+" [FATAL] "+format, v...)
	os.Exit(1)
}

func formatTime() string {
	return time.Now().Format(timeFormat) + " "
}
