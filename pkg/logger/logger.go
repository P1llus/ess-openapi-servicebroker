/*
Package logger is used to define all configuration items and initialize a global
logger object used throughout the application
*/
package logger

import (
	"os"

	"code.cloudfoundry.org/lager"
)

// GetLogger is used to initialize a new Logger object for the application
func GetLogger() lager.Logger {
	logger := lager.NewLogger("ess-servicebroker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	return logger
}
