package dbbackup

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	OperationalDatabaseName = "lifebase"
	DefaultBackupRoot       = "/Volumes/WDRedPlus/LifeBase/backups"
	HourlyDirName           = "hourly"
	DailyDirName            = "daily"
	WeeklyDirName           = "weekly"
	ManualDirName           = "manual"
	DefaultRecentWindow     = 6 * time.Hour
)

var (
	execCommandFn = exec.Command
	mkdirAllFn    = os.MkdirAll
	removeFileFn  = os.Remove
	readDirFn     = os.ReadDir
	copyFileFn    = copyFile
)

func DatabaseNameFromDSN(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(parsed.Path, "/")
	if name == "" {
		return "", fmt.Errorf("database name is empty")
	}
	return name, nil
}

func BackupFileName(dbName string, now time.Time) string {
	return fmt.Sprintf("%s-%s.dump", dbName, now.Format("20060102-150405"))
}

func CategoryDirs(now time.Time) []string {
	dirs := []string{HourlyDirName}
	if now.Hour() == 0 {
		dirs = append(dirs, DailyDirName)
		if now.Weekday() == time.Monday {
			dirs = append(dirs, WeeklyDirName)
		}
	}
	return dirs
}

func Dump(databaseURL, backupDir string, now time.Time, stdout, stderr io.Writer) (string, error) {
	if databaseURL == "" {
		return "", fmt.Errorf("database url is required")
	}
	dbName, err := DatabaseNameFromDSN(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}
	if backupDir == "" {
		backupDir = filepath.Join(DefaultBackupRoot, ManualDirName)
	}
	if err := mkdirAllFn(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	outputPath := filepath.Join(backupDir, BackupFileName(dbName, now))
	cmd := execCommandFn("pg_dump", "--format=custom", "--file", outputPath, databaseURL)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		_ = removeFileFn(outputPath)
		return "", fmt.Errorf("pg_dump: %w", err)
	}
	return outputPath, nil
}

func CopyIntoCategories(sourcePath, root string, categories []string) ([]string, error) {
	copied := make([]string, 0, len(categories))
	base := filepath.Base(sourcePath)
	for _, category := range categories {
		if category == HourlyDirName {
			copied = append(copied, sourcePath)
			continue
		}
		dstDir := filepath.Join(root, category)
		if err := mkdirAllFn(dstDir, 0o755); err != nil {
			return copied, fmt.Errorf("create category dir %s: %w", category, err)
		}
		dst := filepath.Join(dstDir, base)
		if err := copyFileFn(sourcePath, dst); err != nil {
			return copied, fmt.Errorf("copy backup to %s: %w", category, err)
		}
		copied = append(copied, dst)
	}
	return copied, nil
}

func RotateDir(dir, dbName string, keep int) ([]string, error) {
	if keep <= 0 {
		return nil, nil
	}
	entries, err := readDirFn(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, dbName+"-") || !strings.HasSuffix(name, ".dump") {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) <= keep {
		return nil, nil
	}

	removed := make([]string, 0, len(names)-keep)
	for _, name := range names[:len(names)-keep] {
		path := filepath.Join(dir, name)
		if err := removeFileFn(path); err != nil {
			return removed, fmt.Errorf("remove old backup: %w", err)
		}
		removed = append(removed, path)
	}
	return removed, nil
}

func FindLatestRecentBackup(root, dbName string, now time.Time, window time.Duration) (string, time.Time, bool, error) {
	var latestPath string
	var latestTime time.Time

	for _, category := range []string{HourlyDirName, DailyDirName, WeeklyDirName, ManualDirName} {
		dir := filepath.Join(root, category)
		entries, err := readDirFn(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", time.Time{}, false, fmt.Errorf("read backup dir %s: %w", category, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, dbName+"-") || !strings.HasSuffix(name, ".dump") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return "", time.Time{}, false, fmt.Errorf("stat backup file %s: %w", name, err)
			}
			modTime := info.ModTime()
			if latestPath == "" || modTime.After(latestTime) {
				latestPath = filepath.Join(dir, name)
				latestTime = modTime
			}
		}
	}

	if latestPath == "" {
		return "", time.Time{}, false, nil
	}
	if now.Sub(latestTime) > window {
		return latestPath, latestTime, false, nil
	}
	return latestPath, latestTime, true, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		_ = removeFileFn(dst)
		return err
	}
	return out.Sync()
}
