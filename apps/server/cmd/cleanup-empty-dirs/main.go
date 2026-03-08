package main

import (
	"fmt"
	"os"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/fsutil"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	dataRemoved, err := fsutil.RemoveEmptyDirs(cfg.Storage.DataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup data dirs: %v\n", err)
		os.Exit(1)
	}

	thumbRemoved, err := fsutil.RemoveEmptyDirs(cfg.Storage.ThumbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup thumb dirs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("data_empty_dirs_removed=%d\n", dataRemoved)
	fmt.Printf("thumb_empty_dirs_removed=%d\n", thumbRemoved)
}
