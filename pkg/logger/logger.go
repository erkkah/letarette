// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"log"
	"os"
	"strings"
)

// LogWriter is the main logging interface
type LogWriter interface {
	Printf(string, ...interface{})
}

type null struct{}

func (null) Printf(string, ...interface{}) {}

// Debug level log writer
var Debug LogWriter = null{}

// Info level log writer
var Info LogWriter = null{}

// Warning level log writer
var Warning LogWriter = null{}

// Error level log writer
var Error LogWriter = null{}

// LogLevel is the type of all log levels
type LogLevel int

// Debug levels
const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

var levels = map[string]LogLevel{
	"DEBUG":   DEBUG,
	"INFO":    INFO,
	"WARNING": WARNING,
	"ERROR":   ERROR,
}

var currentLevel = INFO

// Level gets the current log level
func Level() LogLevel {
	return currentLevel
}

func init() {
	if level, set := os.LookupEnv("LOG_LEVEL"); set {
		numLevel, found := levels[strings.ToUpper(level)]
		if found {
			currentLevel = numLevel
		}
	}

	switch currentLevel {
	case DEBUG:
		Debug = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags)
		fallthrough
	case INFO:
		Info = log.New(os.Stderr, "[INFO] ", log.LstdFlags)
		fallthrough
	case WARNING:
		Warning = log.New(os.Stderr, "[WARNING] ", log.LstdFlags)
		fallthrough
	case ERROR:
		Error = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	}

}
