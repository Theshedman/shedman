package cmd

import (
	"errors"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunCheck(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	checkCalled := false
	mock.CheckDatabaseFunc = func() error {
		checkCalled = true
		return nil
	}

	err := eng.CheckDatabase()
	if err != nil {
		t.Errorf("CheckDatabase error = %v", err)
	}
	if !checkCalled {
		t.Error("CheckDatabase was not called")
	}

	// Test error prop
	mock.CheckDatabaseFunc = func() error {
		return errors.New("db error")
	}
	err = eng.CheckDatabase()
	if err == nil {
		t.Error("Expected error from CheckDatabase")
	}
}
