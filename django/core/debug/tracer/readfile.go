package tracer

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type fileFrame struct {
	data    []string
	errLine int
}

func (f *fileFrame) Data() []string {
	return f.data
}

func (c *Caller) Read(amountOfLines int) (*fileFrame, error) {
	if !STACKLOGGER_UNSAFE {
		return nil, errors.New("tracer.Caller: unsafe read")
	}
	var ff = &fileFrame{}
	var err error
	ff.data, ff.errLine, err = parseFile(c.File, c.Line, amountOfLines)
	if err != nil {
		return nil, errors.New("tracer.Caller: " + err.Error())
	}
	return ff, nil
}

func (ff *fileFrame) AsString(errPrefix, errSuffix string) string {
	var b strings.Builder
	var totalLen = 0
	for i, line := range ff.data {
		totalLen += len(line)
		//	if i >= ff.errLineStart && i <= ff.errLineEnd {
		//		totalLen += len(errPrefix)
		//		totalLen += len(errSuffix)
		//	}
		if i == ff.errLine {
			totalLen += len(errPrefix)
			totalLen += len(errSuffix)
		}
	}
	b.Grow(totalLen)
	for i, line := range ff.data {
		//	if i >= ff.errLineStart && i <= ff.errLineEnd {
		//		b.WriteString(errPrefix)
		//	}
		//	b.WriteString(line)
		//	if i >= ff.errLineStart && i <= ff.errLineEnd {
		//		b.WriteString(errSuffix)
		//	}
		if i == ff.errLine {
			b.WriteString(errPrefix)
		}
		b.WriteString(line)
		if i == ff.errLine {
			b.WriteString(errSuffix)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func parseFile(path string, line int, amountOfLines int) ([]string, int, error) {
	var err error
	var file *os.File
	var data []string
	line = line - 1 // line is 1-indexed, but we want 0-indexed

	file, err = os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var scanner = bufio.NewScanner(file)
	var i = 0
	var errLine = 0
	// var funcOpen bool
	// var innerOpen = 0
	for scanner.Scan() {
		// if strings.Contains(scanner.Text(), "{") {
		// if !funcOpen {
		// funcOpen = true
		// } else {
		// innerOpen++
		// }
		// }
		// if strings.Contains(scanner.Text(), "}") {
		// if innerOpen > 0 {
		// innerOpen--
		// } else {
		// funcOpen = false
		// }
		// }

		if i >= line-amountOfLines && i <= line+amountOfLines {
			data = append(data, scanner.Text())
			if i == line {
				errLine = len(data) - 1
			}
		}
		i++
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	return data, errLine, nil
}
