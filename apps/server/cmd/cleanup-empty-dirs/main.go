package main

import (
	"fmt"
	"os"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/fsutil"
)

var (
	loadConfigFn      = config.Load
	removeEmptyDirsFn = fsutil.RemoveEmptyDirs
)

func run() error {
	cfg, err := loadConfigFn()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dataRemoved, err := removeEmptyDirsFn(cfg.Storage.DataPath)
	if err != nil {
		return fmt.Errorf("cleanup data dirs: %w", err)
	}

	thumbRemoved, err := removeEmptyDirsFn(cfg.Storage.ThumbPath)
	if err != nil {
		return fmt.Errorf("cleanup thumb dirs: %w", err)
	}

	fmt.Printf("data_empty_dirs_removed=%d\n", dataRemoved)
	fmt.Printf("thumb_empty_dirs_removed=%d\n", thumbRemoved)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
