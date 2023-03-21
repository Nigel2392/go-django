package tracer

import "fmt"

type Caller struct {
	File         string
	Line         int
	FunctionName string
}

func (c *Caller) isAllowed(files []string) bool {
	if files == nil {
		return false
	}
	return contains(files, c.File)
}

func (c *Caller) isDisallowed(files []string) bool {
	if files == nil {
		return false
	}
	return contains(files, c.File)
}

func (c *Caller) String() string {
	return fmt.Sprintf("%s:%d %s", c.File, c.Line, c.FunctionName)
}
