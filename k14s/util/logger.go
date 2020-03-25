package util

import (
	"fmt"

	coreLogger "github.com/k14s/kapp/pkg/kapp/logger"
)

type StdOutLogger struct {
}

var _ coreLogger.Logger = StdOutLogger{}

func NewStdOutLogger() StdOutLogger {
	return StdOutLogger{}
}
func NewTODOLogger() StdOutLogger { return NewStdOutLogger() }

func (l StdOutLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("Error: "+msg, args)
}
func (l StdOutLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("Info: "+msg, args)
}
func (l StdOutLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("Debug: "+msg, args)
}
func (l StdOutLogger) DebugFunc(name string) coreLogger.FuncLogger { return StdOutFuncLogger{} }
func (l StdOutLogger) NewPrefixed(name string) coreLogger.Logger   { return l }

type StdOutFuncLogger struct{}

var _ coreLogger.FuncLogger = StdOutFuncLogger{}

func (l StdOutFuncLogger) Finish() {}
