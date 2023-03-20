package tool

import (
	"fmt"
	"os"

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
	return nil
}

// create directories recursively
func createDir(path string) error {
	var err = os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	return nil
}

func createFile(path string, f []byte) error {
	var dir = httputils.DirFromPath(path)
	var err = createDir(dir)
	if err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return fmt.Errorf("Error creating directory: %w", err)
	}
	err = _writeFile(path, f)
	if err != nil {
		//lint:ignore ST1005 I like capitalized error strings.
		return fmt.Errorf("Error writing file: %w", err)
	}
	return nil
}

func CreateFileStr(path string, f string) error {
	return createFile(path, []byte(f))
}
