package tracer

import "fmt"

type Caller struct {
	File         string
	Line         int
	FunctionName string
}

func (c *Caller) isAllowed(files map[string][]*callerFunc) bool {
	if files == nil {
		return false
	}
	return containsMap(files, c.File)
}

func (c *Caller) isDisallowed(files map[string][]*callerFunc) bool {
	if files == nil {
		return false
	}
	return containsMap(files, c.File)
}

func (c *Caller) String() string {
	return fmt.Sprintf("%s:%d %s", c.File, c.Line, c.FunctionName)
}
