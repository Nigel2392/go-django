package fs_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Nigel2392/go-django/core/fs"
)

func TestFiler(t *testing.T) {
	var _, pathOfTestFile, _, ok = runtime.Caller(0)
	if !ok {
		t.Error("Could not get path of test file")
		return
	}

	var filer = fs.NewFiler(filepath.Join(filepath.Dir(pathOfTestFile), "test_fs"))
	var file, path, err = filer.Create("test.txt")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = file.Write([]byte("Hello World"))
	if err != nil {
		t.Error(err)
		return
	}
	file.Close()

	if filepath.Clean(path) != filepath.Clean(filepath.Join(filer.Base(), "test.txt")) {
		t.Errorf("Expected path to be %s, got %s", filepath.Clean(filepath.Join(filer.Base(), "test.txt")), filepath.Clean(path))
		err = filer.Delete(path)
		if err != nil {
			t.Error(err)
		}
		return
	}

	file2, newpath, err := filer.Create("test.txt")
	if err != nil {
		t.Error(err)
		return
	}

	if filepath.Clean(newpath) == filepath.Clean(path) {
		t.Errorf("Expected path to be different, got %s", newpath)
		return
	}
	file2.Close()

	file3, err := filer.Open("test.txt")
	if err != nil {
		t.Error(err)
		return
	}

	var buf = make([]byte, 11)
	var n, err2 = file3.Read(buf)
	if err2 != nil {
		t.Error(err2)
		return
	}

	if n != 11 {
		t.Errorf("Expected to read 11 bytes, got %d", n)
		return
	}

	if string(buf) != "Hello World" {
		t.Errorf("Expected to read 'Hello World', got '%s'", string(buf))
		return
	}

	file3.Close()

	err = filer.Delete("test.txt")
	if err != nil {
		t.Error(err)
		return
	}

	err = filer.Delete(newpath)
	if err != nil {
		t.Error(err)
		return
	}

}
