package models

import "time"

type UserRow struct {
	User                  User   `json:"user"`
	GroupID               uint64 `json:"group_id"`
	GroupName             string `json:"group_name"`
	GroupDescription      string `json:"group_description"`
	PermissionID          uint64 `json:"permission_id"`
	PermissionName        string `json:"permission_name"`
	PermissionDescription string `json:"permission_description"`
}

type User struct {
	ID              uint64    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	Password        string    `json:"password"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	IsAdministrator bool      `json:"is_administrator"`
	IsActive        bool      `json:"is_active"`
	IsLoggedIn      bool      `json:"is_logged_in"`
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}
