package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const sessionFileMode = 0600

type Session struct {
	UserID       string    `json:"user_id"`
	SessionToken string    `json:"session_token"`
	ServerURL    string    `json:"server_url"`
	CreatedAt    time.Time `json:"created_at"`
}

func StoreSession(session *Session) error {
	if session == nil {
		return errors.New("session is required")
	}
	if session.SessionToken == "" {
		return errors.New("session token is required")
	}

	path, err := sessionPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), sessionFileMode)
}

func sessionPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".rivet", "session.json"), nil
}
