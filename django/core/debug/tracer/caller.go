package tracer

import "fmt"

type Caller struct {
	File         string `json:"file"`
	Line         int    `json:"line"`
	FunctionName string `json:"function_name"`
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
