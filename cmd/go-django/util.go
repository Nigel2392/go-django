package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Nigel2392/go-django/src/core/logger"
)

func copyProjectFiles(files fs.FS, projectPath string, data interface{}) error {
	return fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}
		var name = d.Name()
		var pathToCreateDir = filepath.Dir(
			strings.TrimPrefix(path, "."),
		)

		var isTemplate = strings.HasSuffix(name, ".txt")
		if isTemplate {
			name = strings.TrimSuffix(name, ".txt")
		}

		var pathToCreate = filepath.Join(
			projectPath, pathToCreateDir, name,
		)

		pathToCreate, err = RenderTemplateString(
			pathToCreate, data,
		)
		if err != nil {
			return err
		}

		if d.IsDir() {
			logger.Debugf(
				"Creating directory: %s", pathToCreate,
			)
			err = os.MkdirAll(pathToCreate, 0755)
			if err != nil {
				return err
			}
			return nil
		}

		logger.Debugf(
			"Creating file: %s", pathToCreate,
		)

		newFile, err := os.OpenFile(
			pathToCreate, os.O_CREATE|os.O_WRONLY, 0644,
		)
		if err != nil {
			return err
		}
		defer newFile.Close()

		if isTemplate {
			logger.Debugf(
				"Rendering template: %s", path,
			)
			return RenderTemplateFile(
				files, newFile, path, data,
			)
		}

		file, err := files.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(newFile, file)
		return err
	})
}

func RenderTemplateString(s string, data interface{}) (string, error) {
	var t = template.New("template")
	t.Funcs(funcMap)
	t.Delims("$(", ")")
	t = template.Must(t.Parse(s))
	var b = &bytes.Buffer{}
	err := t.Execute(b, data)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func RenderTemplateFile(files fs.FS, w io.Writer, path string, data interface{}) error {
	var file, err = files.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileContent bytes.Buffer
	_, err = io.Copy(&fileContent, file)
	if err != nil {
		return err
	}

	var t = template.New(path)
	t.Delims("$(", ")")
	t.Funcs(funcMap)
	t = template.Must(
		t.Parse(fileContent.String()),
	)

	return t.Execute(w, data)
}
