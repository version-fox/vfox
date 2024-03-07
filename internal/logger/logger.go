/*
 *    Copyright 2024 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

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
