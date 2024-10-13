package mail_test

import (
	"bytes"
	"os"
	"testing"

	mailpkg "github.com/Nigel2392/go-django/src/core/mail" // Import your mail package here
	"github.com/jordan-wright/email"
)

// Test Console Backend Send
func TestConsoleBackend_Send(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	backend := mailpkg.NewConsoleBackend(&buf)

	// Create a dummy email
	e := &email.Email{
		From: "test@example.com",
		To: []string{
			"recipient@example.com",
			"user@example.com",
		},
		Subject: "Test Email",
		Text:    []byte("This is a test email."),
	}

	// Send the email
	err := backend.Send(e)

	// Ensure no errors occurred
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that the output contains the email
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Subject: Test Email\r\n")) {
		t.Fatalf("expected output to contain 'Subject: Test Email'\\r\\n, got %q", output)
	}
	if !bytes.Contains([]byte(output), []byte("To: <recipient@example.com>, <user@example.com>\r\n")) {
		t.Fatalf("expected output to contain 'To: <recipient@example.com>, <user@example.com>\r\n', got %q", output)
	}

	// Check that the output contains the email body
	if !bytes.Contains([]byte(output), []byte("\r\n\r\nThis is a test email.")) {
		t.Fatalf("expected output to contain '\\r\\n\\r\\This is a test email.', got %q", output)
	}

	// Check that the output contains the email sender
	if !bytes.Contains([]byte(output), []byte("From: <test@example.com>\r\n")) {
		t.Fatalf("expected output to contain 'From: <test@example.com>\\r\\n', got %q", output)
	}
}

// Test Register Backend
func TestRegisterBackend(t *testing.T) {
	// Register a console backend
	mailpkg.Register("testBackend", mailpkg.NewConsoleBackend(os.Stdout))

	// Retrieve the registered backend
	backend := mailpkg.Get("testBackend")

	// Ensure backend is not nil
	if backend == nil {
		t.Fatalf("expected backend to be registered, but got nil")
	}
}

// Test Default Backend Registration
func TestDefaultBackend(t *testing.T) {
	// Ensure the default backend is registered
	defaultBackend := mailpkg.Default()
	if defaultBackend == nil {
		t.Fatalf("expected default backend to be registered, but got nil")
	}
}

// Test Send with Default Backend
func TestSendWithDefaultBackend(t *testing.T) {
	// Create a dummy email
	e := &email.Email{
		From:    "test@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Email",
		Text:    []byte("This is a test email."),
	}

	// Send using the default backend
	err := mailpkg.Send(e)

	// Ensure no errors occurred
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// Test Close Backend
func TestCloseBackend(t *testing.T) {
	// Register an openable backend
	mailpkg.Register("testBackend", &MockOpenableBackend{})

	backend := mailpkg.Get("testBackend")
	if backend == nil {
		t.Fatalf("expected backend to be registered, but got nil")
	}

	// Ensure backend is open
	if !backend.(mailpkg.OpenableEmailBackend).IsOpen() {
		t.Fatalf("expected backend to be open, but got closed")
	}

	// Close the backend
	err := mailpkg.Close("testBackend")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// Mock implementation of OpenableEmailBackend for testing
type MockOpenableBackend struct {
	isOpen bool
}

func (m *MockOpenableBackend) Open() error {
	m.isOpen = true
	return nil
}

func (m *MockOpenableBackend) IsOpen() bool {
	return m.isOpen
}

func (m *MockOpenableBackend) Close() error {
	m.isOpen = false
	return nil
}

func (m *MockOpenableBackend) Send(e *email.Email) error {
	return nil
}
