package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Source  string // Path of directory containing source bdfr posts.
	DestDir string // Destination directory to save media and processed posts to.
	Verbose bool   // True for verbose output.
	Jobs    int    // Number of jobs to run in parallel.
}

func parseArgs() (*Config, error) {
	verbose := flag.Bool("v", false, "verbose output")
	jobs := flag.Int("j", 1, "jobs")

	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) < 1 {
		return nil, fmt.Errorf("missing required argument: source")
	}
	source := flag.Args()[0]

	if len(flag.Args()) < 2 {
		return nil, fmt.Errorf("missing required argument: dest_dir")
	}
	destDir := flag.Args()[1]

	return &Config{
		Source:  source,
		DestDir: destDir,
		Verbose: *verbose,
		Jobs:    *jobs,
	}, nil
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [option]... <source> <dest_dir>\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(flag.CommandLine.Output(), "Scrapes media links from a bdfr archive.\n")
	flag.PrintDefaults()
}
