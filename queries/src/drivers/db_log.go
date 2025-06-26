package drivers

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/logger"
)

var LOG_SQL_QUERIES = true

type canLogContextKey struct{}

func canLogSQL(ctx context.Context) bool {
	var value = ctx.Value(canLogContextKey{})
	if value == nil {
		return LOG_SQL_QUERIES
	}

	var boolean, ok = value.(bool)
	if !ok {
		return LOG_SQL_QUERIES
	}

	return boolean
}

func wasLogged(level logger.LogLevel) bool {
	return logger.GetLevel() <= level
}

func LogSQL(ctx context.Context, from string, err error, query string, args ...any) (logged bool) {
	if !canLogSQL(ctx) {
		return false
	}
	if err != nil {
		logger.Errorf("[%s.Query]: %s: %s %v", from, err.Error(), query, args)
		return wasLogged(logger.ERR)
	}
	logger.Debugf("[%s.Query]: %s %v", from, query, args)
	return wasLogged(logger.DBG)
}

func SetLogSQL(ctx context.Context, log bool) context.Context {
	return context.WithValue(ctx, canLogContextKey{}, log)
}
