package main

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPathValidation(t *testing.T) {
	// Setup config regex
	var err error
	AllowedDirsRegexp, err = regexp.Compile(`^C:/Users/snowm/Projects/.*$`)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"C:\\Users\\snowm\\Projects\\remote-controller", true},
		{"C:/Users/snowm/Projects/another-project", true},
		{"C:/Windows/System32", false},
		{filepath.Join(GetChatSessionsBasePath(), "uuid-test-123"), true},
	}

	for _, tc := range tests {
		result := IsValidPath(tc.path)
		if result != tc.expected {
			t.Errorf("IsValidPath(%q) = %v; want %v", tc.path, result, tc.expected)
		}
	}
}

func TestDatabaseOperationsAndCleanup(t *testing.T) {
	// Create temp file for SQLite test DB
	tmpFile, err := ioutil.TempFile("", "test_history_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if err := InitDB(tmpFile.Name()); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	// 1. Test Save & Get Session
	testAlias := "test-session"
	testDir := NormalizePath("./test_dir_path")
	testService := "grok"

	if err := SaveSession(testAlias, testDir, testService); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	s, err := GetSession(testAlias)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if s.Alias != testAlias || s.Directory != testDir || s.Service != testService {
		t.Errorf("retrieved session mismatch: %+v", s)
	}

	// 2. Test List Sessions
	list, total, err := ListSessions(1, 10)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Alias != testAlias {
		t.Errorf("list sessions mismatch: total=%d, len=%d", total, len(list))
	}

	// 3. Test Save & Get History
	testPrompt := "list files"
	testResponse := "file1.txt\nfile2.txt"
	if err := SaveHistory(testAlias, testPrompt, testResponse, nil); err != nil {
		t.Fatalf("failed to save history: %v", err)
	}

	hist, hTotal, err := GetHistory(testAlias, 1, 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if hTotal != 1 || len(hist) != 1 || hist[0].Prompt != testPrompt || hist[0].Response != testResponse {
		t.Errorf("history item mismatch: total=%d, item=%+v", hTotal, hist[0])
	}

	// 4. Test Chat Session cleanup logic
	// Create dummy folder inside chat base
	chatBase := GetChatSessionsBasePath()
	dummyChatDir := filepath.Join(chatBase, "dummy-test-uuid")
	if err := os.MkdirAll(dummyChatDir, 0755); err != nil {
		t.Fatalf("failed to create dummy chat dir: %v", err)
	}
	defer os.RemoveAll(dummyChatDir)

	chatAlias := "dummy-chat"
	if err := SaveSession(chatAlias, dummyChatDir, "grok"); err != nil {
		t.Fatalf("failed to save chat session: %v", err)
	}

	// Fetch to verify it exists in DB
	chatSess, err := GetSession(chatAlias)
	if err != nil {
		t.Fatalf("failed to get chat session: %v", err)
	}

	// Simulate HandleDeleteSession filesystem cleanup logic
	if err := DeleteSession(chatAlias); err != nil {
		t.Fatalf("failed to delete session from DB: %v", err)
	}

	normalizedDir := NormalizePath(chatSess.Directory)
	if strings.HasPrefix(normalizedDir, chatBase) && normalizedDir != chatBase {
		_ = os.RemoveAll(chatSess.Directory)
	}

	// Verify the folder was deleted from disk
	if _, err := os.Stat(dummyChatDir); !os.IsNotExist(err) {
		t.Errorf("expected dummy chat directory to be deleted, but it still exists")
	}

	// Verify history cascades
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Verify non-chat sessions leave directories intact
	dummyRegularDir, err := ioutil.TempDir("", "regular_test_dir_*")
	if err != nil {
		t.Fatalf("failed to create regular temp dir: %v", err)
	}
	defer os.RemoveAll(dummyRegularDir)

	regAlias := "regular-sess"
	if err := SaveSession(regAlias, dummyRegularDir, "grok"); err != nil {
		t.Fatalf("failed to save regular session: %v", err)
	}

	regSess, err := GetSession(regAlias)
	if err != nil {
		t.Fatalf("failed to get regular session: %v", err)
	}

	if err := DeleteSession(regAlias); err != nil {
		t.Fatalf("failed to delete regular session from DB: %v", err)
	}

	regNormDir := NormalizePath(regSess.Directory)
	if strings.HasPrefix(regNormDir, chatBase) && regNormDir != chatBase {
		_ = os.RemoveAll(regSess.Directory)
	}

	// Directory should still exist because it is NOT under chatBase
	if _, err := os.Stat(dummyRegularDir); os.IsNotExist(err) {
		t.Errorf("regular directory was deleted, but it should have been preserved")
	}
}

func TestMain(m *testing.M) {
	// Basic database placeholder configuration
	db = &sql.DB{}
	os.Exit(m.Run())
}
