package logger

import (
	"os"

	"code.cloudfoundry.org/lager"
)

func GetLogger() lager.Logger {
	logger := lager.NewLogger("ess-servicebroker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	return logger
}
