package fileutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// FileExists returns true if a file or directory with the given path exists.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// IsDir returns true if a directory with the given path exists.
func IsDir(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && info.IsDir()
}

// RecursiveCopyIf conditionally copies all files rooted at srcDir to their
// equivalent relative path rooted at dstDir. For each file, it performs a copy
// if pred returns true. For each directory, it descends if pred retruns true.
func RecursiveCopyIf(srcDir string, dstDir string, pred func(i os.FileInfo) bool) error {
	absDst, err := filepath.Abs(dstDir)
	if err != nil {
		return err
	}

	var iter func(relPath string) error
	iter = func(relPath string) error {
		fullSrc, err := filepath.Abs(srcDir)
		if err != nil {
			return err
		}

		fullDst, err := filepath.Abs(dstDir)
		if err != nil {
			return err
		}

		if relPath != "" {
			fullSrc = filepath.Join(fullSrc, relPath)
			fullDst = filepath.Join(fullDst, relPath)
		}

		// Don't copy anything out of the destination directory (in case
		// destination is a subdirectory of source).
		if strings.HasPrefix(fullSrc, absDst) {
			log.Debugf("skipping copy to avoid infinite recursion: src=%s", fullSrc)
			return nil
		}

		info, err := os.Stat(fullSrc)
		if err != nil {
			return err
		}

		if !pred(info) {
			return nil
		}

		if info.IsDir() {
			err := os.MkdirAll(fullDst, info.Mode())
			if err != nil {
				return err
			}

			entries, err := os.ReadDir(fullSrc)
			if err != nil {
				return err
			}

			for _, e := range entries {
				err := iter(filepath.Join(relPath, e.Name()))
				if err != nil {
					return err
				}
			}

			return nil
		}

		log.Debugf("copying: %s --> %s", fullSrc, fullDst)

		b, err := os.ReadFile(fullSrc)
		if err != nil {
			return err
		}

		err = os.WriteFile(fullDst, b, fs.ModePerm)
		if err != nil {
			return err
		}

		return nil
	}

	return iter("")
}
