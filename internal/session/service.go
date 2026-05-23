// Package session handles session persistence and message storage.
package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spenc/savant-cli/internal/db"
)

// Session represents a conversation session.
type Session struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	ProviderName string    `json:"provider_name"`
	ModelName    string    `json:"model_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Message represents a stored message.
type Message struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"session_id"`
	Role        string          `json:"role"`
	Content     string          `json:"content"`
	ToolCalls   json.RawMessage `json:"tool_calls,omitempty"`
	ToolResults json.RawMessage `json:"tool_results,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Service provides session CRUD operations.
type Service struct {
	db *db.DB
}

// NewService creates a new session service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new session and returns it.
func (s *Service) Create(ctx context.Context, id, title, provider, model string) (*Session, error) {
	now := time.Now()
	_, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO sessions (id, title, provider_name, model_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, title, provider, model, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	return &Session{
		ID:           id,
		Title:        title,
		ProviderName: provider,
		ModelName:    model,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Get retrieves a session by ID.
func (s *Service) Get(ctx context.Context, id string) (*Session, error) {
	var sess Session
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, title, message_count, input_tokens, output_tokens, cost,
		        provider_name, model_name, created_at, updated_at
		 FROM sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.Title, &sess.MessageCount, &sess.InputTokens,
		&sess.OutputTokens, &sess.Cost, &sess.ProviderName, &sess.ModelName,
		&sess.CreatedAt, &sess.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &sess, nil
}

// List returns all sessions, most recent first.
func (s *Service) List(ctx context.Context, limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, title, message_count, input_tokens, output_tokens, cost,
		        provider_name, model_name, created_at, updated_at
		 FROM sessions ORDER BY updated_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.MessageCount,
			&sess.InputTokens, &sess.OutputTokens, &sess.Cost,
			&sess.ProviderName, &sess.ModelName, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// UpdateStats updates the token counts and cost for a session.
func (s *Service) UpdateStats(ctx context.Context, id string, inputTokens, outputTokens int, cost float64) error {
	_, err := s.db.Conn().ExecContext(ctx,
		`UPDATE sessions SET input_tokens = input_tokens + ?, output_tokens = output_tokens + ?,
		        cost = cost + ?, updated_at = ?
		 WHERE id = ?`,
		inputTokens, outputTokens, cost, time.Now(), id,
	)
	return err
}

// Delete removes a session and all its messages.
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.db.Conn().ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// AddMessage stores a message in a session.
func (s *Service) AddMessage(ctx context.Context, msg Message) error {
	toolCallsJSON, _ := json.Marshal(msg.ToolCalls)
	toolResultsJSON, _ := json.Marshal(msg.ToolResults)

	_, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO messages (id, session_id, role, content, tool_calls, tool_results, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content,
		string(toolCallsJSON), string(toolResultsJSON), msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	// Update session message count
	_, err = s.db.Conn().ExecContext(ctx,
		`UPDATE sessions SET message_count = message_count + 1, updated_at = ? WHERE id = ?`,
		time.Now(), msg.SessionID,
	)
	return err
}

// GetMessages retrieves all messages for a session.
func (s *Service) GetMessages(ctx context.Context, sessionID string) ([]Message, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, session_id, role, content, tool_calls, tool_results, created_at
		 FROM messages WHERE session_id = ? ORDER BY created_at ASC`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var toolCalls, toolResults sql.NullString
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content,
			&toolCalls, &toolResults, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if toolCalls.Valid {
			msg.ToolCalls = json.RawMessage(toolCalls.String)
		}
		if toolResults.Valid {
			msg.ToolResults = json.RawMessage(toolResults.String)
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// RecordFileChange records a file modification for rollback capability.
func (s *Service) RecordFileChange(ctx context.Context, sessionID, filePath, changeType, oldContent, newContent string) error {
	_, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO file_changes (id, session_id, file_path, change_type, old_content, new_content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("%s-%d", filePath, time.Now().UnixNano()),
		sessionID, filePath, changeType, oldContent, newContent, time.Now(),
	)
	return err
}

// GetFileChanges retrieves file changes for a session.
func (s *Service) GetFileChanges(ctx context.Context, sessionID string) ([]FileChange, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, session_id, file_path, change_type, old_content, new_content, created_at
		 FROM file_changes WHERE session_id = ? ORDER BY created_at ASC`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get file changes: %w", err)
	}
	defer rows.Close()

	var changes []FileChange
	for rows.Next() {
		var fc FileChange
		var oldContent, newContent sql.NullString
		if err := rows.Scan(&fc.ID, &fc.SessionID, &fc.FilePath, &fc.ChangeType,
			&oldContent, &newContent, &fc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan file change: %w", err)
		}
		if oldContent.Valid {
			fc.OldContent = oldContent.String
		}
		if newContent.Valid {
			fc.NewContent = newContent.String
		}
		changes = append(changes, fc)
	}
	return changes, rows.Err()
}

// FileChange represents a recorded file modification.
type FileChange struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	FilePath   string    `json:"file_path"`
	ChangeType string    `json:"change_type"`
	OldContent string    `json:"old_content,omitempty"`
	NewContent string    `json:"new_content,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// SessionSummary is a compact representation of a session for extraction.
type SessionSummary struct {
	ID           string
	Title        string
	MessageCount int
	UpdatedAt    time.Time
}

// MessageSummary is a compact representation of a message.
type MessageSummary struct {
	Role    string
	Content string
}

// ListEligibleForExtraction returns sessions eligible for skill extraction.
func (s *Service) ListEligibleForExtraction(ctx context.Context, minMessages int, idleDuration time.Duration) ([]SessionSummary, error) {
	cutoff := time.Now().Add(-idleDuration)
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, title, message_count, updated_at
		 FROM sessions
		 WHERE message_count >= ? AND updated_at <= ?
		 ORDER BY updated_at DESC`,
		minMessages, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("list eligible sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionSummary
	for rows.Next() {
		var sess SessionSummary
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.MessageCount, &sess.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// GetMessageSummaries returns compact message summaries for a session.
func (s *Service) GetMessageSummaries(ctx context.Context, sessionID string) ([]MessageSummary, error) {
	msgs, err := s.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var summaries []MessageSummary
	for _, m := range msgs {
		summaries = append(summaries, MessageSummary{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return summaries, nil
}
