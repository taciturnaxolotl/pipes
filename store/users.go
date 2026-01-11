package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         string
	IndikoSub  string
	Username   string
	Name       string
	Email      string
	Photo      string
	URL        string
	Role       string
	CreatedAt  int64
	UpdatedAt  int64
}

type Session struct {
	ID           string
	UserID       string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	CreatedAt    int64
}

func (db *DB) CreateUser(indikoSub, username, name, email, photo, url string) (*User, error) {
	now := time.Now().Unix()
	user := &User{
		ID:        uuid.New().String(),
		IndikoSub: indikoSub,
		Username:  username,
		Name:      name,
		Email:     email,
		Photo:     photo,
		URL:       url,
		Role:      "user",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := db.Exec(`
		INSERT INTO users (id, indiko_sub, username, name, email, photo, url, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.IndikoSub, user.Username, user.Name, user.Email, user.Photo, user.URL, user.Role, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (db *DB) GetUserByIndikoSub(indikoSub string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, indiko_sub, username, name, email, photo, url, role, created_at, updated_at
		FROM users
		WHERE indiko_sub = ?
	`, indikoSub).Scan(&user.ID, &user.IndikoSub, &user.Username, &user.Name, &user.Email, &user.Photo, &user.URL, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	return user, nil
}

func (db *DB) GetUserByID(id string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, indiko_sub, username, name, email, photo, url, role, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id).Scan(&user.ID, &user.IndikoSub, &user.Username, &user.Name, &user.Email, &user.Photo, &user.URL, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	return user, nil
}

func (db *DB) UpdateUser(user *User) error {
	user.UpdatedAt = time.Now().Unix()

	_, err := db.Exec(`
		UPDATE users
		SET username = ?, name = ?, email = ?, photo = ?, url = ?, updated_at = ?
		WHERE id = ?
	`, user.Username, user.Name, user.Email, user.Photo, user.URL, user.UpdatedAt, user.ID)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (db *DB) CreateSession(userID, accessToken, refreshToken string, expiresAt int64) (*Session, error) {
	session := &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now().Unix(),
	}

	_, err := db.Exec(`
		INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.AccessToken, session.RefreshToken, session.ExpiresAt, session.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return session, nil
}

func (db *DB) GetSessionByID(id string) (*Session, error) {
	session := &Session{}
	err := db.QueryRow(`
		SELECT id, user_id, access_token, refresh_token, expires_at, created_at
		FROM sessions
		WHERE id = ?
	`, id).Scan(&session.ID, &session.UserID, &session.AccessToken, &session.RefreshToken, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	return session, nil
}

func (db *DB) DeleteSession(id string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (db *DB) DeleteExpiredSessions() error {
	now := time.Now().Unix()
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", now)
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}
