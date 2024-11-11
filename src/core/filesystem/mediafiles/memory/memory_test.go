package memory_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
)

func TestBackend_SaveAndExists(t *testing.T) {
	backend := memory.NewBackend(5)
	content := "Hello, World!"
	fileName := "testfile.txt"

	// Test saving a file
	savedName, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}
	if savedName != fileName {
		t.Errorf("expected filename to match %v, got %v", fileName, savedName)
	}

	// Test file existence
	exists, err := backend.Exists(fileName)
	if err != nil {
		t.Errorf("expected no error while checking existence, got: %v", err)
	}
	if !exists {
		t.Error("expected file to exist, but it does not")
	}
}

func TestBackend_Delete(t *testing.T) {
	backend := memory.NewBackend(5)
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
	backend := memory.NewBackend(5)
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
}

func TestBackend_ListDir(t *testing.T) {
	backend := memory.NewBackend(5)
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
}

func TestBackend_Stat(t *testing.T) {
	backend := memory.NewBackend(5)
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
}

func TestBackend_Open(t *testing.T) {
	backend := memory.NewBackend(5)
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
}

func TestBackend_GetValidName(t *testing.T) {
	backend := memory.NewBackend(5)
	fileName := "testfile.txt"
	content := "Content for valid name test"

	// Save a file
	_, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	// Get a valid name for an existing file
	validName := backend.GetValidName(fileName)
	if validName == fileName {
		t.Errorf("expected a modified valid name for an existing file, got same name: %v", validName)
	}
	if !strings.HasPrefix(validName, "testfile_") {
		t.Errorf("expected valid name to have modified prefix, got: %v", validName)
	}
}

func TestBackend_GenerateFilename(t *testing.T) {
	backend := memory.NewBackend(5)
	fileName := "testfile.txt"
	content := "Content for filename generation test"

	// Save a file
	_, err := backend.Save(fileName, strings.NewReader(content))
	if err != nil {
		t.Errorf("expected no error while saving file, got: %v", err)
	}

	// Generate filename for a new save
	generatedName := backend.GenerateFilename(fileName)
	if generatedName == fileName {
		t.Errorf("expected generated filename to be unique for existing file, got same name: %v", generatedName)
	}
}
