package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/logutils"
	cli "github.com/integrii/flaggy"
	"github.com/spf13/afero"

	"github.com/wormi4ok/mdmv/internal"
)

var version = "dev"

func init() {
	cli.SetName("mdmv")
	cli.SetDescription("Move markdown files")
	cli.SetVersion(version)

	cli.DefaultParser.AdditionalHelpPrepend = "https://github.com/wormi4ok/mdmv"
}

func main() {
	var src, dest string
	var debug bool
	cli.AddPositionalValue(&src, "src", 1, true, "Source path")
	cli.AddPositionalValue(&dest, "dest", 2, true, "Target destination")
	cli.Bool(&debug, "v", "verbose", "enable debug logging")
	cli.Parse()

	setLogLevel(debug)

	fs := afero.NewOsFs()
	err := mv(fs, src, dest)
	if err != nil {
		log.Printf("[ERROR] %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}

var (
	errNoFilesFound  = errors.New("no markdown files found")
	errNoFilesToMove = errors.New("no files to move")
	errMoveMultiple  = errors.New("specify an existing directory as a destination for multiple markdown files")
	errWrongTemplate = errors.New("incorrect template in the destination path")
)

// mv moves files from source to destination
// It relies on the filesystem abstraction provided by afero library.
func mv(fs afero.Fs, src, dest string) error {
	files, err := findFiles(fs, src)
	if err != nil {
		return err
	}

	mdFiles := parseFiles(fs, files)

	// Collect paths to clean up before we move files
	dirs := cleanupList(mdFiles)

	err = moveFiles(fs, mdFiles, dest)
	if err != nil {
		return err
	}

	return cleanUp(fs, dirs)
}

// findFiles accepts a filename, directories and glob patterns as an input
// and returns a slice files matching the search criteria.
func findFiles(fs afero.Fs, path string) (files []string, err error) {
	// If input is a directory, find all *.md files
	if isDir(fs, path) {
		path = filepath.FromSlash(path + "/*.md")
	}

	if strings.Contains(path, "*") {
		files, err = afero.Glob(fs, path)
		if files == nil {
			err = errNoFilesFound
		}

		return
	}

	files = []string{path}

	return
}

// parseFiles converts every filepath to an internal representation
// suitable for further operations.
func parseFiles(fs afero.Fs, files []string) []*internal.File {
	ff := make([]*internal.File, 0, len(files))
	for _, file := range files {
		md, err := internal.NewFile(fs, file)
		if err != nil {
			log.Printf("Skipping %s: %v", file, err)
			continue
		}
		ff = append(ff, md)
	}
	return ff
}

// cleanupList prepares a list of existing directories
// that might be empty after move operation.
func cleanupList(files []*internal.File) []string {
	dirs := make(map[string]struct{})

	// Find potential dirs to remove
	for _, file := range files {
		dirs[path.Dir(file.Path)] = struct{}{}
		for _, attachment := range file.Attachments {
			attDir := filepath.Join(path.Dir(file.Path), path.Dir(attachment))
			dirs[attDir] = struct{}{}
		}
	}

	// Turn the existing ones into a slice
	paths := make([]string, 0, len(dirs))
	for k := range dirs {
		paths = append(paths, k)
	}
	return paths
}

// moveFiles decides where to move each file based on the destination definition.
//
// Destination is a string representing on of the following:
// * filename - only if the source is also a single file
// * directory - moves all source files as is to the new directory
// * template path - build a custom destination using the template or each file
//
// A template like `path/%title%/index.md` will get replaced with the title
// of the markdown file (not the filename, but the header text inside the file).
func moveFiles(fs afero.Fs, files []*internal.File, dest string) error {
	if len(files) == 0 {
		return errNoFilesToMove
	}

	tplFound, err := isTemplate(dest)
	if err != nil {
		return err
	}

	if tplFound {
		for _, file := range files {
			err := file.Move(replaceTemplates(dest, file.Title))
			if err != nil {
				return err
			}
		}
		return nil
	}

	if isDir(fs, dest) {
		for _, file := range files {
			err := file.MoveToDir(dest)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if len(files) > 1 {
		return errMoveMultiple
	}

	if err := files[0].Move(dest); err != nil {
		return fmt.Errorf("failed to move a file: %w", err)
	}
	return nil
}

// cleanUp removes directories that are empty, starting from the directories deep in the filesystem tree.
func cleanUp(fs afero.Fs, keys []string) error {
	// Put deep directories on top of the list
	// To remove nested empty dirs first and parent dirs after
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	// Remove empty dirs from the source path
	for _, dir := range keys {
		if exists, _ := afero.DirExists(fs, dir); !exists {
			continue
		}
		if empty, err := afero.IsEmpty(fs, dir); empty && err == nil {
			err := fs.Remove(dir)
			if err != nil {
				return fmt.Errorf("failed to clean up a dir: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check if dir is empty: %w", err)
		}
	}

	return nil
}

// replaceTemplates replaces supported templates.
func replaceTemplates(tpl string, title string) string {
	escaped := strings.ReplaceAll(title, string(filepath.Separator), "_")
	return strings.Replace(tpl, "%title%", escaped, 1)
}

// isTemplate checks if the string is a template.
func isTemplate(path string) (bool, error) {
	if !strings.Contains(path, "%") {
		return false, nil
	}

	if !strings.Contains(path, "%title%") {
		return false, errWrongTemplate
	}

	return true, nil
}

// isDir checks if a given file path is a directory.
func isDir(fs afero.Fs, path string) bool {
	info, err := fs.Stat(path)
	return err == nil && info.IsDir()
}

// setLogLevel configures LevelFilter to print debug messages
// if debug argument is true, otherwise sets the log level to WARN.
func setLogLevel(debug bool) {
	var logLevel logutils.LogLevel = "WARN"

	if debug {
		logLevel = "DEBUG"
	}

	log.SetOutput(&logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logLevel,
		Writer:   os.Stderr,
	})
}
