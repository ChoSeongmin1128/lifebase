package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/dbbackup"
)

var (
	loadConfigFn                   = config.Load
	nowFn                          = time.Now
	backupDumpFn                   = dbbackup.Dump
	copyIntoCategoriesFn           = dbbackup.CopyIntoCategories
	rotateBackupDirFn              = dbbackup.RotateDir
	runFn                          = run
	exitFn                         = os.Exit
	stderrWriter         io.Writer = os.Stderr
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
		return fmt.Errorf("automatic backup only supports %s database, got %s", dbbackup.OperationalDatabaseName, dbName)
	}

	root := os.Getenv("DB_BACKUP_ROOT")
	if root == "" {
		root = dbbackup.DefaultBackupRoot
	}

	now := nowFn()
	hourlyDir := filepath.Join(root, dbbackup.HourlyDirName)
	backupPath, err := backupDumpFn(cfg.Database.URL, hourlyDir, now, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	copied, err := copyIntoCategoriesFn(backupPath, root, dbbackup.CategoryDirs(now))
	if err != nil {
		return err
	}
	fmt.Printf("automatic_backup_paths=%s\n", joinPaths(copied))

	limits := map[string]int{
		dbbackup.HourlyDirName: 14,
		dbbackup.DailyDirName:  14,
		dbbackup.WeeklyDirName: 8,
	}
	for _, category := range []string{dbbackup.HourlyDirName, dbbackup.DailyDirName, dbbackup.WeeklyDirName} {
		removed, err := rotateBackupDirFn(filepath.Join(root, category), dbName, limits[category])
		if err != nil {
			return err
		}
		if len(removed) > 0 {
			fmt.Printf("rotated_%s=%s\n", category, joinPaths(removed))
		}
	}
	return nil
}

func joinPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return strings.Join(paths, ",")
}

func main() {
	if err := runFn(); err != nil {
		fmt.Fprintf(stderrWriter, "%v\n", err)
		exitFn(1)
	}
}
