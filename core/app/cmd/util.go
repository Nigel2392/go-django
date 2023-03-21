package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/core/httputils"
)

func _writeFile(path string, f []byte) error {
	// write the env_template to a new .env file.
	var _, err = os.Stat(path)
	if os.IsNotExist(err) {
		// File does not exist, create it.
		var file, err = os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = file.Write(f)
		if err != nil {
			return err
		}
	}
	return err
}

// create directories recursively
func createDir(path string) error {
	var err = os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	return nil
}

func createFile(path string, f []byte) {
	var dir = httputils.DirFromPath(path)
	var err = createDir(dir)
	if err != nil {
		panic(err)
	}
	err = _writeFile(path, f)
	if err != nil {
		panic(err)
	}
}

type Tag struct {
	Name    string `json:"name"`
	Integer int    `json:"-"`
}

type Tags []Tag

func (t Tags) Len() int {
	return len(t)
}

func DecodeTags(data io.Reader) Tags {
	var tags = Tags{}
	decoder := json.NewDecoder(data)
	err := decoder.Decode(&tags)
	if err != nil {
		panic(err)
	}
	return tags
}

func (t Tags) initInts() {
	for i := 0; i < len(t); i++ {
		var tag = strings.TrimPrefix(t[i].Name, "v")
		var tag_parts = strings.Split(tag, ".")
		var newTag = strings.Join(tag_parts, "")
		for len(newTag) < 4 {
			newTag += "0"
		}
		var newTagInt, err = strconv.Atoi(newTag)
		if err != nil {
			panic(errors.New("Could not create tags: " + err.Error()))
		}
		t[i].Integer = newTagInt
	}
}

func (t Tags) Descending() {
	t.initInts()
	for i := 0; i < len(t); i++ {
		for j := i + 1; j < len(t); j++ {
			if t[i].Integer < t[j].Integer {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
}

func (t Tags) Ascending() {
	t.initInts()
	for i := 0; i < len(t); i++ {
		for j := i + 1; j < len(t); j++ {
			if t[i].Integer > t[j].Integer {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
}

func (t Tags) Latest() Tag {
	t.initInts()
	t.Descending()
	return t[0]
}
