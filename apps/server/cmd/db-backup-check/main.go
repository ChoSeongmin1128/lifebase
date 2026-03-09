package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/dbbackup"
)

var (
	loadConfigFn             = config.Load
	nowFn                    = time.Now
	recentBackupFn           = dbbackup.FindLatestRecentBackup
	runFn                    = run
	exitFn                   = os.Exit
	stderrWriter   io.Writer = os.Stderr
)

func run() error {
	cfg, err := loadConfigFn()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Database.URL == "" {
		return fmt.Errorf("database url is required")
	}

	dbName, err := dbbackup.DatabaseNameFromDSN(cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	if dbName != dbbackup.OperationalDatabaseName {
		fmt.Printf("skip_recent_backup_check_for=%s\n", dbName)
		return nil
	}

	root := os.Getenv("DB_BACKUP_ROOT")
	if root == "" {
		root = dbbackup.DefaultBackupRoot
	}

	path, modTime, ok, err := recentBackupFn(root, dbName, nowFn(), dbbackup.DefaultRecentWindow)
	if err != nil {
		return err
	}
	if !ok {
		if path == "" {
			return fmt.Errorf("recent operational backup not found under %s", root)
		}
		return fmt.Errorf("recent operational backup missing: latest=%s modified_at=%s", path, modTime.Format(time.RFC3339))
	}
	fmt.Printf("recent_backup_path=%s\n", path)
	return nil
}

func main() {
	if err := runFn(); err != nil {
		fmt.Fprintf(stderrWriter, "%v\n", err)
		exitFn(1)
	}
}
