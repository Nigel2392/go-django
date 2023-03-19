package cmd

import (
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
