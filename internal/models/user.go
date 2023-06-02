package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string     `json:"id" db:"id"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt" db:"deleted_at"`
	Name      string     `json:"name" db:"name"`
}

func NewUser(now time.Time, name string) *User {
	return &User{
		ID:        newID("user"),
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
		Name:      name,
	}
}

type UserCredential struct {
	ID        UserCredentialID    `json:"id" db:"id"`
	CreatedAt time.Time           `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time           `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time          `json:"deletedAt" db:"deleted_at"`
	UserID    string              `json:"userID" db:"user_id"`
	Data      *UserCredentialData `json:"data" db:"data"`
}

func NewUserCredential(now time.Time, userID string, id UserCredentialID, data *UserCredentialData) *UserCredential {
	return &UserCredential{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
		UserID:    userID,
		Data:      data,
	}
}

type UserCredentialID string

func UserCredentialGitHub(username string) UserCredentialID {
	return UserCredentialID("github:" + username)
}

func (i UserCredentialID) Name() string {
	kind, data, found := strings.Cut(string(i), ":")
	if !found {
		return string(i)
	}

	switch kind {
	case "github":
		name := data
		return name

	default:
		return string(i)
	}
}

type UserCredentialData struct {
	KeyFingerprint string `json:"keyFingerprint,omitempty"`
}

func (d *UserCredentialData) Scan(val any) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}
func (d *UserCredentialData) Value() (driver.Value, error) {
	return json.Marshal(d)
}

type TokenClaims struct {
	Username string `json:"username,omitempty"`
	jwt.RegisteredClaims
}
