package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/dbbackup"
)

var (
	loadConfigFn           = config.Load
	nowFn                  = time.Now
	backupDumpFn           = dbbackup.Dump
	runFn                  = run
	exitFn                 = os.Exit
	stderrWriter io.Writer = os.Stderr
)

func run() (string, error) {
	cfg, err := loadConfigFn()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	if cfg.Database.URL == "" {
		return "", fmt.Errorf("database url is required")
	}

	backupDir := os.Getenv("DB_BACKUP_DIR")
	if backupDir == "" {
		backupDir = filepath.Join(dbbackup.DefaultBackupRoot, dbbackup.ManualDirName)
	}

	outputPath, err := backupDumpFn(cfg.Database.URL, backupDir, nowFn(), os.Stdout, os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Printf("backup_path=%s\n", outputPath)
	return outputPath, nil
}

func main() {
	if _, err := runFn(); err != nil {
		fmt.Fprintf(stderrWriter, "%v\n", err)
		exitFn(1)
	}
}
