package messages

import (
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func RequestMessageContext(c ctx.ContextWithRequest) {
	var backend = Backend(c.Request())
	if backend == nil {
		return
	}

	messages, _ := backend.Get()
	c.Set("messages", messages)
	c.Set("DEFAULT_MESSAGE_LEVELS", app.Tags)

	logger.NameSpace(MESSAGES_NAMESPACE).Debugf(
		"Messages context processor: %T, stored messages: %d",
		backend, len(messages),
	)

	except.AssertNil(
		backend.Clear(),
		500, "messages: error clearing messages",
	)
}
