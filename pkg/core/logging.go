package core

import (
	"time"

	"github.com/theshedman/shedman/pkg/log"
)

func (e *Engine) logTransaction(action string, pkgs []string, success bool) {
	if e == nil || e.config == nil || !e.config.Logging.Enabled {
		return
	}

	path := e.config.Logging.File
	if path == "" {
		return
	}

	logger := log.New(path)
	_ = logger.Log(log.Transaction{
		Timestamp: time.Now(),
		Action:    action,
		Packages:  pkgs,
		Success:   success,
	})
}
