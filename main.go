package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccollins476ad/bdfrscrape/fileutil"
	log "github.com/sirupsen/logrus"
)

func printFatalError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

func main() {
	cfg, err := parseArgs()
	if err != nil {
		printFatalError(err)
		flag.Usage()
		os.Exit(1)
	}

	if cfg.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	// Collect filenames of posts in source directory.
	var filenames []string
	filepath.WalkDir(cfg.Source, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			filenames = append(filenames, d.Name())
		}
		return nil
	})

	// Copy non-post files to destination directory. These are files that won't
	// get processed, but which the posts reference, and thus need a manual
	// copy (reddit media).
	err = fileutil.RecursiveCopyIf(cfg.Source, cfg.DestDir, func(info os.FileInfo) bool {
		return info.IsDir() || !strings.HasSuffix(info.Name(), ".json")
	})
	if err != nil {
		printFatalError(err)
		os.Exit(2)
	}

	err = processFiles(context.Background(), cfg, filenames)
	if err != nil {
		printFatalError(err)
		os.Exit(3)
	}
}
