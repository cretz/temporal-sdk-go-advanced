package testutil_test

import (
	"fmt"

	"go.temporal.io/sdk/log"
)

type simpleLoggerImpl struct{}

var simpleLogger log.Logger = simpleLoggerImpl{}

func simpleLog(level, msg string, keyvals ...interface{}) {
	fmt.Println(append([]interface{}{level, msg}, keyvals...)...)
}

func (simpleLoggerImpl) Debug(msg string, keyvals ...interface{}) {
	simpleLog("DEBUG", msg, keyvals...)
}

func (simpleLoggerImpl) Info(msg string, keyvals ...interface{}) {
	simpleLog("INFO ", msg, keyvals...)
}

func (simpleLoggerImpl) Warn(msg string, keyvals ...interface{}) {
	simpleLog("WARN ", msg, keyvals...)
}

func (simpleLoggerImpl) Error(msg string, keyvals ...interface{}) {
	simpleLog("ERROR", msg, keyvals...)
}
