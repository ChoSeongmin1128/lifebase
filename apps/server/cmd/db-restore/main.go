package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"lifebase/internal/shared/config"
)

var (
	loadConfigFn            = config.Load
	execCommandFn           = exec.Command
	runFn                   = run
	exitFn                  = os.Exit
	stderrWriter  io.Writer = os.Stderr
)

func run(args []string) error {
	cfg, err := loadConfigFn()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Database.URL == "" {
		return fmt.Errorf("database url is required")
	}

	fs := flag.NewFlagSet("db-restore", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var filePath string
	fs.StringVar(&filePath, "file", "", "pg_dump backup file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if filePath == "" {
		return fmt.Errorf("backup file path is required")
	}

	cmd := execCommandFn("pg_restore", "--clean", "--if-exists", "--no-owner", "--dbname", cfg.Database.URL, filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore: %w", err)
	}

	fmt.Printf("restored_from=%s\n", filePath)
	return nil
}

func main() {
	if err := runFn(os.Args[1:]); err != nil {
		fmt.Fprintf(stderrWriter, "%v\n", err)
		exitFn(1)
	}
}
