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

// LogSQL logs the SQL query and its arguments if logging is enabled in the context.
//
// It logs the query with the source of the query (from) and the error if it exists.
//
// If the error is not nil, it logs an error message; otherwise, it logs a debug message.
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

// SetLogSQL sets the logging flag for SQL queries in the context.
//
// It allows you to enable or disable SQL query logging for the current context.
func SetLogSQLContext(ctx context.Context, log bool) context.Context {
	return context.WithValue(ctx, canLogContextKey{}, log)
}

// LogSQLScope takes a context, a bool and a function, and returns a new context
// with SQL logging enabled or disabled based on the bool value.
//
// It returns three values:
// - A new context with the SQL logging flag set.
// - A possible error returned by the function.
// - A boolean indicating whether the SQL query was logged or not.
func LogSQLScope(ctx context.Context, log bool, fn func(context.Context) error) (context.Context, error, bool) {
	var newCtx = SetLogSQLContext(ctx, log)
	var err = fn(newCtx)
	var logged = canLogSQL(newCtx) && wasLogged(logger.DBG)

	// probably should do this only when query could be logged? right?
	if err != nil && log && logged {
		logger.Errorf("Error in LogSQLScope: %v", err)
	}

	return newCtx, err, logged
}
