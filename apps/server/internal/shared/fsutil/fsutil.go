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

var (
	removeDirFn = os.Remove
	statPathFn  = os.Stat
	walkDirFn   = filepath.WalkDir
	relPathFn   = filepath.Rel
	dirPathFn   = filepath.Dir
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

		err := removeDirFn(current)
		switch {
		case err == nil:
		case os.IsNotExist(err):
		case isDirNotEmptyError(err):
			return nil
		default:
			return err
		}

		parent := dirPathFn(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func RemoveEmptyDirs(root string) (int, error) {
	root = filepath.Clean(root)
	if _, err := statPathFn(root); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	dirs := make([]string, 0, 32)
	if err := walkDirFn(root, func(path string, d fs.DirEntry, err error) error {
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
		err := removeDirFn(dir)
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
	rel, err := relPathFn(root, target)
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
