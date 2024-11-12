package fs_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
)

func TestBackend_SaveAndExists(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	content := "Hello, World!"
	fileName := "testfile.txt"

	// Test saving a file
	t.Logf("Saving file %v", fileName)
	savedName, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}
	if savedName != fileName {
		t.Errorf("expected filename to match %v, got %v", fileName, savedName)
	}

	t.Log("Checking file existence")
	// Test file existence
	exists, err := backend.Exists(fileName)
	if err != nil {
		t.Errorf("expected no error while checking existence, got: %v", err)
	}
	if !exists {
		t.Error("expected file to exist, but it does not")
	}

	t.Logf("Deleting file %v", fileName)
	if err := backend.Delete(fileName); err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}
}

func TestBackend_SaveAndExists_Rename(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	content := "Hello, World!"
	fileName := "testfile.txt"

	t.Logf("Saving file (1) %v", fileName)
	// Test saving a file
	fileNameNew1, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	t.Logf("Saving file (2) %v", fileName)
	fileNameNew2, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	if fileNameNew1 == fileNameNew2 {
		t.Errorf("expected different filenames, got same: %v", fileNameNew1)
	}

	if fileNameNew1 != fileName && fileNameNew2 != fileName {
		t.Errorf("expected one of the filenames to match the original, got %v and %v", fileNameNew1, fileNameNew2)
	}

	t.Log("Checking file existence (1)")
	// Test file existence
	exists, err := backend.Exists(fileNameNew1)
	if err != nil {
		t.Errorf("expected no error while checking existence, got: %v", err)
	}
	if !exists {
		t.Error("expected file to exist, but it does not")
	}

	t.Log("Checking file existence (2)")
	// Test file existence
	exists, err = backend.Exists(fileNameNew2)
	if err != nil {
		t.Errorf("expected no error while checking existence, got: %v", err)
	}
	if !exists {
		t.Error("expected file to exist, but it does not")
	}

	t.Logf("Deleting file (1) %v", fileName)
	if err := backend.Delete(fileNameNew1); err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}

	t.Logf("Deleting file (2) %v", fileName)
	if err := backend.Delete(fileNameNew2); err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}
}

func TestBackend_Delete(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	fileName := "testfile.txt"
	content := "Temporary content"

	// Save a file to delete
	_, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	// Delete the file
	err = backend.Delete(fileName)
	if err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}

	// Ensure the file no longer exists
	exists, err := backend.Exists(fileName)
	if err != nil {
		t.Errorf("expected no error while checking existence after delete, got: %v", err)
	}
	if exists {
		t.Error("expected file to be deleted, but it still exists")
	}
}

func TestBackend_GetAvailableName(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	fileName := "testfile.txt"
	content := "Sample content"

	// Save a file
	_, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	// Test that it provides an alternative name for an existing file
	availableName, err := backend.GetAvailableName(fileName, 5, 100)
	if err != nil {
		t.Errorf("expected no error while generating alternative name, got: %v", err)
	}
	if availableName == fileName {
		t.Errorf("expected a different name for an existing file, got same name: %v", availableName)
	}

	if err := backend.Delete(fileName); err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}
}

func TestBackend_ListDir(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	fileNames := []string{"dir1/file1.txt", "dir1/file2.txt", "dir2/file3.txt"}

	// Save files
	for _, name := range fileNames {
		_, err := backend.Save(name, strings.NewReader("Content"))
		if err != nil {
			t.Errorf("expected no error while saving files for ListDir test, got: %v", err)
		}
	}

	// List directory
	files, err := backend.ListDir("dir1")
	if err != nil {
		t.Errorf("expected no error while listing directory, got: %v", err)
	}

	expectedFiles := []string{"file1.txt", "file2.txt"}
	if len(files) != len(expectedFiles) {
		t.Errorf("expected %d files, got %d", len(expectedFiles), len(files))
	}
	for i, file := range files {
		if file != expectedFiles[i] {
			t.Errorf("expected file %v, got %v", expectedFiles[i], file)
		}
	}

	for _, name := range fileNames {
		if err := backend.Delete(name); err != nil {
			t.Errorf("expected no error while deleting file, got: %v", err)
		}
	}
}

func TestBackend_Stat(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	fileName := "testfile.txt"
	content := "File for stat test"

	// Save a file
	_, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	// Check file stats
	header, err := backend.Stat(fileName)
	if err != nil {
		t.Errorf("expected no error while getting file stat, got: %v", err)
	}
	if header == nil {
		t.Error("expected non-nil file header")
	}

	if err := backend.Delete(fileName); err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}
}

func TestBackend_Open(t *testing.T) {
	backend := fs.NewBackend("./test", 5)
	fileName := "testfile.txt"
	content := "Content to read"

	// Save a file
	_, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	// Open the file and read the content
	obj, err := backend.Open(fileName)
	if err != nil {
		t.Errorf("expected no error while opening file, got: %v", err)
	}
	if obj == nil {
		t.Fatal("expected non-nil stored object")
	}

	file, err := obj.Open()
	if err != nil {
		t.Errorf("expected no error while opening file content, got: %v", err)
	}

	// Read file content
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		t.Errorf("expected no error while reading file content, got: %v", err)
	}
	if buf.String() != content {
		t.Errorf("expected file content to match '%s', got '%s'", content, buf.String())
	}

	if err := file.Close(); err != nil {
		t.Errorf("expected no error while closing file, got: %v", err)
	}

	if err := backend.Delete(fileName); err != nil {
		t.Errorf("expected no error while deleting file, got: %v", err)
	}
}
