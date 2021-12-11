package internal

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
)

var titleRe = regexp.MustCompile(`#+(.*)`)
var imageRe = regexp.MustCompile(`!\[.*]\((.+)\)`)

// File is an abstraction representing a Markdown file
// All fields are populated by NewFile constructor.
type File struct {
	fs   afero.Fs
	Path string

	Title       string
	Attachments []string
}

// NewFile creates File structure from the file in the path.
func NewFile(fs afero.Fs, path string) (*File, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open a source file: %w", err)
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Printf("[ERROR] Failed to close the file: %s", path)
		}
	}()

	var title string
	var attachments []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if title == "" {
			title = strings.TrimLeft(titleRe.FindString(scanner.Text()), " #")
		}
		image := imageRe.FindStringSubmatch(scanner.Text())
		if len(image) > 1 {
			attachments = append(attachments, image[1])
		}
	}

	return &File{
		fs: fs,

		Title:       title,
		Path:        path,
		Attachments: attachments,
	}, nil
}

// Move file to a new destination.
func (f *File) Move(dest string) error {
	if exists, err := afero.Exists(f.fs, dest); exists && err == nil {
		file := filepath.Base(dest)
		log.Printf("[WARN] file already exists %s", file)
	}

	if err := ensureDir(f.fs, dest); err != nil {
		return err
	}

	if err := f.fs.Rename(f.Path, dest); err != nil {
		return fmt.Errorf("failed to move the file: %w", err)
	}
	defer func() { f.Path = dest }()

	dir := filepath.Dir(f.Path)
	if dir == filepath.Dir(dest) || len(f.Attachments) == 0 {
		return nil
	}

	for _, attachment := range f.Attachments {
		destDir := filepath.Dir(dest)
		attachmentDest := filepath.Join(destDir, attachment)

		if err := ensureDir(f.fs, attachmentDest); err != nil {
			return err
		}

		if err := f.fs.Rename(filepath.Join(dir, attachment), attachmentDest); err != nil {
			return fmt.Errorf("failed to move the attachment: %w", err)
		}
	}

	return nil
}

// MoveToDir moves file to a new directory.
func (f *File) MoveToDir(dirName string) error {
	dest := f.uniqueName(filepath.Join(dirName, filepath.Base(f.Path)))
	if err := f.fs.Rename(f.Path, dest); err != nil {
		return fmt.Errorf("failed to move a file: %w", err)
	}
	if len(f.Attachments) == 0 {
		return nil
	}

	for _, attachment := range f.Attachments {
		src := filepath.Join(filepath.Dir(f.Path), attachment)
		dest := filepath.Join(dirName, attachment)

		if exists, err := afero.Exists(f.fs, src); !exists && err == nil {
			log.Printf("[WARN] attachment is missing: %s", src)
			continue
		}

		if err := ensureDir(f.fs, dest); err != nil {
			return err
		}

		if err := f.fs.Rename(filepath.Join(filepath.Dir(f.Path), attachment), dest); err != nil {
			return fmt.Errorf("failed to move an attachment: %w", err)
		}
	}

	return nil
}

// ensureDir creates a directory tree for a destination file on a filesystem if it doesn't exist
func ensureDir(fs afero.Fs, dest string) error {
	dir := filepath.Dir(dest)
	exists, err := afero.DirExists(fs, dir)
	if err != nil {
		return err
	}
	if !exists {
		if err = fs.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create a directory: %w", err)
		}
	}

	return nil
}

// uniqueName returns a destination path with appended unique
// suffix if the file with the same name already exists on the filesystem.
func (f *File) uniqueName(dest string) string {
	i := 1
	for exists, err := afero.Exists(f.fs, dest); exists && err == nil; i++ {
		if i == 1 {
			log.Printf("[WARN] File with the same name already exists: %s", dest)
		}

		file := filepath.Base(strings.ToLower(dest))
		ext := filepath.Ext(file)
		name := regexp.MustCompile(`(?mU)(.*)_*\d*$`).FindStringSubmatch(strings.TrimSuffix(file, ext))
		file = fmt.Sprintf("%s_%d%s", name[1], i, ext)
		dest = filepath.Join(filepath.Dir(dest), file)
		exists, err = afero.Exists(f.fs, dest)
	}

	return dest
}
