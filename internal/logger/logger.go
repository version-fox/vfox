package logger

import "fmt"

type LoggerLevel int

const (
	ErrorLevel LoggerLevel = iota
	InfoLevel
	DebugLevel
)

var globalLevel = InfoLevel

func SetLevel(_level LoggerLevel) {
	globalLevel = _level
}

func Log(level LoggerLevel, args ...interface{}) {
	if globalLevel >= level {
		fmt.Println(args...)
	}
}

func Logf(level LoggerLevel, message string, args ...interface{}) {
	if globalLevel >= level {
		fmt.Printf(message, args...)
	}
}

func Error(message string) {
	Log(ErrorLevel, message)
}

func Errorf(message string, args ...interface{}) {
	Logf(ErrorLevel, message, args...)
}

func Info(message string) {
	Log(InfoLevel, message)
}

func Infof(message string, args ...interface{}) {
	Logf(InfoLevel, message, args...)
}

func Debug(args ...interface{}) {
	Log(DebugLevel, args...)
}

func Debugf(message string, args ...interface{}) {
	Logf(DebugLevel, message, args...)
}
