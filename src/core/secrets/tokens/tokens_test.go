package tokens_test

import (
	"context"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/secrets/tokens"
)

type User struct {
	ID        int
	Email     string
	Name      string
	LastLogin time.Time
}

func (u *User) TokenParts() ([]any, error) {
	return []any{u.ID, u.Email, u.LastLogin}, nil
}

func TestTokenValid(t *testing.T) {
	// Create a new user
	user := &User{
		ID:        1,
		Email:     "test@example.com",
		Name:      "Test User",
		LastLogin: time.Now(),
	}

	var generator = tokens.NewResetTokenGenerator(
		"sha256", "myapp.tokens.ResetTokenGenerator", []byte("supersecretkey"), nil, time.Hour*24, nil,
	)

	// Generate a token for the user
	var ctx = context.Background()
	token, err := generator.MakeToken(ctx, user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	t.Logf("Generated token: %s", token)

	// Check the validity of the token
	valid, err := generator.CheckToken(ctx, user, token)
	if err != nil {
		t.Fatalf("failed to check token: %v", err)
	}

	if !valid {
		t.Errorf("token is not valid")
	}
}
