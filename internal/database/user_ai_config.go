package database

import (
	"errors"
	"strings"
	"time"
)

// UserAIConfig stores a per-user LLM configuration so that each registered user
// can bring their own API key / base URL / model instead of sharing the platform
// global AI channel.
type UserAIConfig struct {
	UserID              string
	Provider            string
	APIKey              string
	BaseURL             string
	Model               string
	MaxTotalTokens      int
	MaxCompletionTokens int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Configured reports whether the user has supplied enough to run the agent on
// their own credentials: an API key, a base URL and a model name.
func (c *UserAIConfig) Configured() bool {
	return c != nil &&
		strings.TrimSpace(c.APIKey) != "" &&
		strings.TrimSpace(c.BaseURL) != "" &&
		strings.TrimSpace(c.Model) != ""
}

func (db *DB) initUserAIConfigTable() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_ai_configs (
			user_id TEXT PRIMARY KEY,
			provider TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			base_url TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			max_total_tokens INTEGER NOT NULL DEFAULT 0,
			max_completion_tokens INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES rbac_users(id) ON DELETE CASCADE
		);`)
	return err
}

// GetUserAIConfig returns the per-user AI config for the given user, or
// sql.ErrNoRows when the user has not configured one yet.
func (db *DB) GetUserAIConfig(userID string) (*UserAIConfig, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	row := db.QueryRow(`
		SELECT user_id, provider, api_key, base_url, model, max_total_tokens, max_completion_tokens, created_at, updated_at
		FROM user_ai_configs WHERE user_id = ?`, userID)
	var c UserAIConfig
	var createdAt, updatedAt string
	if err := row.Scan(&c.UserID, &c.Provider, &c.APIKey, &c.BaseURL, &c.Model, &c.MaxTotalTokens, &c.MaxCompletionTokens, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	c.CreatedAt = parseDBTime(createdAt)
	c.UpdatedAt = parseDBTime(updatedAt)
	return &c, nil
}

// UpsertUserAIConfig creates or replaces the per-user AI config.
func (db *DB) UpsertUserAIConfig(userID, provider, apiKey, baseURL, model string, maxTotalTokens, maxCompletionTokens int) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user id is required")
	}
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO user_ai_configs (user_id, provider, api_key, base_url, model, max_total_tokens, max_completion_tokens, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			provider=excluded.provider,
			api_key=excluded.api_key,
			base_url=excluded.base_url,
			model=excluded.model,
			max_total_tokens=excluded.max_total_tokens,
			max_completion_tokens=excluded.max_completion_tokens,
			updated_at=excluded.updated_at
	`, userID, strings.TrimSpace(provider), apiKey, strings.TrimSpace(baseURL), strings.TrimSpace(model), maxTotalTokens, maxCompletionTokens, now, now)
	return err
}
