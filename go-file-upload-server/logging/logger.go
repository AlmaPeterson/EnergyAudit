package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

type Logger struct {
	ServiceName string
}

func NewLogger() Logger {
	return Logger{}
}

func (l Logger) Log(logLevel LogLevel, message string) {
	if logLevel == LogLevelDebug {
		if os.Getenv("DEBUG_LOGGING") == "false" {
			return
		}
	}

	if l.ServiceName == "" {
		log.Print("No service name set for logger")
		return
	}

	numSpacesToAdd := 7 - len(logLevel)
	spaces := ""
	if numSpacesToAdd > 0 {
		spaces = strings.Repeat(" ", numSpacesToAdd)
	}

	color := ""

	switch logLevel {
	case LogLevelDebug:
		color = "lightgray"
	case LogLevelInfo:
		color = "blue"
	case LogLevelWarning:
		color = "yellow"
	case LogLevelError:
		color = "red"
	}

	svcName := l.ServiceName
	logLvlString := WrapColor(color, "%v", logLevel)
	if color == "" {
		logLvlString = fmt.Sprintf("%v", logLevel)
	}

	log.Printf("%v%v - %v - %v", logLvlString, spaces, svcName, message)
}

func (l Logger) Info(message string) {
	l.Log(LogLevelInfo, message)
}

func (l Logger) Infof(format string, args ...any) {
	l.Log(LogLevelInfo, fmt.Sprintf(format, args...))
}

func (l Logger) Debug(message string) {
	l.Log(LogLevelDebug, message)
}

func (l Logger) Debugf(format string, args ...any) {
	l.Log(LogLevelDebug, fmt.Sprintf(format, args...))
}

func (l Logger) Warning(message string) {
	l.Log(LogLevelWarning, message)
}

func (l Logger) Warningf(format string, args ...any) {
	l.Log(LogLevelWarning, fmt.Sprintf(format, args...))
}

func (l Logger) Error(message string) {
	l.Log(LogLevelError, message)
}

func (l Logger) Errorf(format string, args ...any) {
	l.Log(LogLevelError, fmt.Sprintf(format, args...))
}

func WrapColor(color, format string, parts ...any) string {
	strng := fmt.Sprintf(format, parts...)

	_, ok := Colors[color]
	if ok {
		color = Colors[color]
	}

	return fmt.Sprintf("\033[38;2;%vm%v\033[0m", color, strng)
}

var Colors map[string]string = map[string]string{
	"green":       "0;150;50",
	"lightgreen":  "100;255;150",
	"red":         "255;0;0",
	"blue":        "0;0;255",
	"yellow":      "255;255;0",
	"lightgray":   "200;200;200",
	"lightblue":   "150;150;255",
	"lightred":    "255;150;150",
	"lightyellow": "255;255;150",
	"brown":       "139;69;19",
}
