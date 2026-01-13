package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunKeyring(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	// Mock flags
	initCalled := false
	refreshCalled := false
	addCalled := ""
	removeCalled := ""
	importCalled := ""
	listCalled := false

	// Match field names in MockBackend (verb-object convention mostly)
	mock.InitKeyringFunc = func() error {
		initCalled = true
		return nil
	}
	mock.RefreshKeysFunc = func() error {
		refreshCalled = true
		return nil
	}
	mock.ListKeysFunc = func() ([]string, error) {
		listCalled = true
		return []string{"key1", "key2"}, nil
	}
	mock.AddKeyFunc = func(key string) error {
		addCalled = key
		return nil
	}
	mock.RemoveKeyFunc = func(key string) error {
		removeCalled = key
		return nil
	}
	mock.ImportKeyFunc = func(path string) error {
		importCalled = path
		return nil
	}

	var buf bytes.Buffer

	// Test Init
	if err := RunKeyringInit(eng, &buf); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if !initCalled {
		t.Error("Init not called")
	}
	if !strings.Contains(buf.String(), "Initializing keyring...") {
		t.Error("Missing init output")
	}
	buf.Reset()

	// Test Refresh
	if err := RunKeyringRefresh(eng, &buf); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if !refreshCalled {
		t.Error("Refresh not called")
	}
	if !strings.Contains(buf.String(), "Refreshing keys...") {
		t.Error("Missing refresh output")
	}
	buf.Reset()

	// Test List
	if err := RunKeyringList(eng, &buf); err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if !listCalled {
		t.Error("List not called")
	}
	if !strings.Contains(buf.String(), "key1") || !strings.Contains(buf.String(), "key2") {
		t.Error("Missing listed keys")
	}
	buf.Reset()

	// Test Add
	if err := RunKeyringAdd(eng, &buf, "abcd"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if addCalled != "abcd" {
		t.Error("Add not called with abcd")
	}
	if !strings.Contains(buf.String(), "Adding key abcd...") {
		t.Error("Missing add output")
	}
	buf.Reset()

	// Test Remove
	if err := RunKeyringRemove(eng, &buf, "1234"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if removeCalled != "1234" {
		t.Error("Remove not called with 1234")
	}
	if !strings.Contains(buf.String(), "Removing key 1234...") {
		t.Error("Missing remove output")
	}
	buf.Reset()

	// Test Import
	if err := RunKeyringImport(eng, &buf, "/tmp/key.gpg"); err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if importCalled != "/tmp/key.gpg" {
		t.Error("Import not called with path")
	}
	if !strings.Contains(buf.String(), "Importing key from /tmp/key.gpg...") {
		t.Error("Missing import output")
	}
}
