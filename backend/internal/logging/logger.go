package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Component string `json:"component,omitempty"`
	Message   string `json:"message"`
}

type Logger struct {
	mu     sync.Mutex
	file   *os.File
	path   string
	recent []Entry
}

const maxRecentEntries = 2000

var (
	globalMu sync.RWMutex
	global   = &Logger{}
)

func Setup(dir string) (*Logger, error) {
	if strings.TrimSpace(dir) == "" {
		dir = filepath.Join("data", "logs")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "backend.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	logger := &Logger{
		file:   file,
		path:   path,
		recent: loadRecentEntries(path, maxRecentEntries),
	}
	globalMu.Lock()
	global = logger
	globalMu.Unlock()
	return logger, nil
}

func Default() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func (l *Logger) Path() string {
	if l == nil {
		return ""
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.path
}

func (l *Logger) Tail(limit int) []Entry {
	if limit <= 0 {
		limit = 200
	}
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.recent) <= limit {
		return append([]Entry{}, l.recent...)
	}
	return append([]Entry{}, l.recent[len(l.recent)-limit:]...)
}

func (l *Logger) log(level, component, format string, args ...any) {
	if l == nil {
		return
	}
	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     strings.ToLower(strings.TrimSpace(level)),
		Component: strings.TrimSpace(component),
		Message:   fmt.Sprintf(format, args...),
	}
	text := formatTextEntry(entry)
	raw, _ := json.Marshal(entry)

	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(os.Stdout, text)
	if l.file != nil {
		_, _ = l.file.Write(append(raw, '\n'))
	}
	l.recent = append(l.recent, entry)
	if len(l.recent) > maxRecentEntries {
		l.recent = append([]Entry{}, l.recent[len(l.recent)-maxRecentEntries:]...)
	}
}

func formatTextEntry(entry Entry) string {
	base := fmt.Sprintf("%s [%s]", entry.Timestamp, strings.ToUpper(entry.Level))
	if entry.Component != "" {
		base += " " + entry.Component
	}
	return base + " " + entry.Message
}

func loadRecentEntries(path string, limit int) []Entry {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	items := make([]Entry, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		items = append(items, entry)
		if len(items) > limit {
			items = append([]Entry{}, items[len(items)-limit:]...)
		}
	}
	return items
}

func Infof(component, format string, args ...any) {
	Default().log("info", component, format, args...)
}

func Warnf(component, format string, args ...any) {
	Default().log("warn", component, format, args...)
}

func Errorf(component, format string, args ...any) {
	Default().log("error", component, format, args...)
}

func Debugf(component, format string, args ...any) {
	Default().log("debug", component, format, args...)
}
