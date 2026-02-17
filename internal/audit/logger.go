package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
)

// Logger writes structured audit log entries to JSON files.
type Logger struct {
	logDir  string
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewLogger creates a new audit logger that writes to the given directory.
func NewLogger(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("creating audit log dir: %w", err)
	}

	filename := fmt.Sprintf("audit-%s.jsonl", time.Now().Format("2006-01-02T15-04-05"))
	path := filepath.Join(logDir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening audit log: %w", err)
	}

	return &Logger{
		logDir:  logDir,
		file:    f,
		encoder: json.NewEncoder(f),
	}, nil
}

// Log writes a single audit entry.
func (l *Logger) Log(entry types.AuditEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	_ = l.encoder.Encode(entry)
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// ReadSession reads all audit entries for a specific session.
func ReadSession(logDir, sessionID string) ([]types.AuditEntry, error) {
	files, err := filepath.Glob(filepath.Join(logDir, "audit-*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("listing audit logs: %w", err)
	}

	var entries []types.AuditEntry
	for _, f := range files {
		fileEntries, err := readLogFile(f, sessionID)
		if err != nil {
			continue
		}
		entries = append(entries, fileEntries...)
	}

	return entries, nil
}

func readLogFile(path, sessionID string) ([]types.AuditEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []types.AuditEntry
	dec := json.NewDecoder(strings.NewReader(string(data)))

	for dec.More() {
		var entry types.AuditEntry
		if err := dec.Decode(&entry); err != nil {
			continue
		}
		if sessionID == "" || entry.SessionID == sessionID {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
