package messages

type MessageTag string

const (
	DEBUG   MessageTag = "debug"
	INFO    MessageTag = "info"
	SUCCESS MessageTag = "success"
	WARNING MessageTag = "warning"
	ERROR   MessageTag = "error"

	LEVEL_DEBUG   uint = 10
	LEVEL_INFO    uint = 20
	LEVEL_SUCCESS uint = 30
	LEVEL_WARNING uint = 40
	LEVEL_ERROR   uint = 50
)

type MessageTags struct {
	Debug   MessageTag
	Info    MessageTag
	Success MessageTag
	Warning MessageTag
	Error   MessageTag
}

var DefaultTags = MessageTags{
	Debug:   DEBUG,
	Info:    INFO,
	Success: SUCCESS,
	Warning: WARNING,
	Error:   ERROR,
}

var TagLevels = map[MessageTag]uint{
	DEBUG:   LEVEL_DEBUG,
	INFO:    LEVEL_INFO,
	SUCCESS: LEVEL_SUCCESS,
	WARNING: LEVEL_WARNING,
	ERROR:   LEVEL_ERROR,
}

var LevelTags = map[uint]MessageTag{
	LEVEL_DEBUG:   DEBUG,
	LEVEL_INFO:    INFO,
	LEVEL_SUCCESS: SUCCESS,
	LEVEL_WARNING: WARNING,
	LEVEL_ERROR:   ERROR,
}
