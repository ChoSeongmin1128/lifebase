package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

func PruneEmptyParents(root, start string) error {
	root = filepath.Clean(root)
	current := filepath.Clean(start)

	for {
		if current == root {
			return nil
		}
		if !isWithinRoot(root, current) {
			return nil
		}

		err := os.Remove(current)
		switch {
		case err == nil:
		case os.IsNotExist(err):
		case isDirNotEmptyError(err):
			return nil
		default:
			return err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func RemoveEmptyDirs(root string) (int, error) {
	root = filepath.Clean(root)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	dirs := make([]string, 0, 32)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() || path == root {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	}); err != nil {
		return 0, err
	}

	sort.SliceStable(dirs, func(i, j int) bool {
		return depth(dirs[i]) > depth(dirs[j])
	})

	removed := 0
	for _, dir := range dirs {
		err := os.Remove(dir)
		switch {
		case err == nil:
			removed++
		case os.IsNotExist(err), isDirNotEmptyError(err):
			continue
		default:
			return removed, err
		}
	}

	return removed, nil
}

func isWithinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func isDirNotEmptyError(err error) bool {
	return errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST)
}

func depth(path string) int {
	return strings.Count(filepath.Clean(path), string(os.PathSeparator))
}
