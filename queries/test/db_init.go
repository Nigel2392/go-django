package queries_test

import (
	"flag"
	"os"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func init() {
	testing.Init()

	flag.Parse()

	var which, db = testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(django.Configure(settings))

	logger.Debugf("Using %s database for queries tests", which)

	if !testing.Verbose() {
		logger.SetLevel(logger.WRN)
	}

}
