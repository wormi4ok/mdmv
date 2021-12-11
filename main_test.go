package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestMv(t *testing.T) {
	testFs := newTmpFS(t)

	src := "sample.md"
	dest := "dest.md"

	copyFile(t, testFs, filepath.Join("testdata", src), src)

	err := mv(testFs, src, dest)
	require.NoError(t, err)

	require.False(t, fileExists(testFs, src), "Source file still exists")
	require.True(t, fileExists(testFs, dest), "Destination file exists")
}

func TestMvToDir(t *testing.T) {
	file := "sample.md"
	attachment := "images/sample.png"
	dest := "folder"

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", file), file)
	copyFile(t, testFs, filepath.Join("testdata", attachment), attachment)
	err := testFs.Mkdir(dest, os.ModePerm)
	failOn(t, err, "create directory")

	err = mv(testFs, file, dest)
	require.NoError(t, err)

	require.False(t, fileExists(testFs, file), "Source file exists")
	require.True(t, fileExists(testFs, filepath.Join(dest, file)), "File moved")

	require.False(t, fileExists(testFs, attachment), "Source attachment path exists")
	require.True(t, fileExists(testFs, filepath.Join(dest, attachment)), "Attachment moved")
}

func TestMvFromDir(t *testing.T) {
	first := "sample.md"
	second := "second.md"
	attachment := "images/sample.png"
	src := "test"
	dest := "folder"

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", first), filepath.Join(src, first))
	copyFile(t, testFs, filepath.Join("testdata", second), filepath.Join(src, second))
	copyFile(t, testFs, filepath.Join("testdata", attachment), filepath.Join(src, attachment))

	// At first, test if it fails with non-existing directory
	err := mv(testFs, src, dest)
	require.Error(t, err)

	err = testFs.Mkdir(dest, os.ModePerm)
	failOn(t, err, "create directory")

	err = mv(testFs, src, dest)
	require.NoError(t, err)

	require.False(t, fileExists(testFs, filepath.Join(src, first)), "First source file exists")
	require.False(t, fileExists(testFs, filepath.Join(src, second)), "Second source file exists")
	require.False(t, dirExists(testFs, src), "Source directory exists")

	require.True(t, fileExists(testFs, filepath.Join(dest, first)), "First file moved")
	require.True(t, fileExists(testFs, filepath.Join(dest, second)), "Second file moved")

	require.False(t, fileExists(testFs, attachment), "Source attachment path exists")
	require.True(t, fileExists(testFs, filepath.Join(dest, attachment)), "Attachment moved")
	require.False(t, dirExists(testFs, filepath.Dir(attachment)), "Source attachment dir exists")
}

func TestMvGlob(t *testing.T) {
	sample := "sample.md"
	second := "second.md"
	attachment := "images/sample.png"
	folder1 := "one folder"
	folder2 := "second folder"

	pattern := "*/*.md"
	dest := "third folder"

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", sample), filepath.Join(folder1, sample))
	copyFile(t, testFs, filepath.Join("testdata", attachment), filepath.Join(folder1, attachment))
	copyFile(t, testFs, filepath.Join("testdata", second), filepath.Join(folder2, second))
	err := testFs.Mkdir(dest, os.ModePerm)
	failOn(t, err, "create directory")

	err = mv(testFs, pattern, dest)
	require.NoError(t, err)

	require.False(t, fileExists(testFs, filepath.Join(folder1, sample)), "Source file exists")
	require.False(t, fileExists(testFs, filepath.Join(folder1, attachment)), "Attachment exists")

	require.False(t, dirExists(testFs, folder1), "First source directory exists")
	require.False(t, dirExists(testFs, folder2), "Second source directory exists")
	require.False(t, dirExists(testFs, filepath.Dir(attachment)), "Source attachment dir exists")

	require.True(t, fileExists(testFs, filepath.Join(dest, sample)), "First file moved")
	require.True(t, fileExists(testFs, filepath.Join(dest, second)), "Second file moved")
	require.True(t, fileExists(testFs, filepath.Join(dest, attachment)), "Attachment moved")
}

func TestMvTemplateTitle(t *testing.T) {
	wantTitle := "Sample file for testing purpose"
	src := "sample.md"
	attachment := "images/sample.png"
	dest := "%title%/README.md"

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", src), src)
	copyFile(t, testFs, filepath.Join("testdata", attachment), attachment)

	err := mv(testFs, src, dest)
	require.NoError(t, err)

	require.False(t, fileExists(testFs, src), "Source file exists")
	require.False(t, fileExists(testFs, attachment), "Attachment exists")

	require.True(t, fileExists(testFs, filepath.Join(wantTitle, "README.md")), "File moved")
	require.True(t, fileExists(testFs, filepath.Join(wantTitle, attachment)), "Attachment moved")
	require.False(t, dirExists(testFs, filepath.Dir(attachment)), "Source attachment dir exists")
}

func TestMvUnicode(t *testing.T) {
	wantTitle := "Меры безопасности для защиты сервера _ petrashov.ru"
	src := "unicode.md"
	dest := "%title%/index.md"

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", src), src)

	err := mv(testFs, src, dest)
	require.NoError(t, err)

	require.False(t, fileExists(testFs, src), "Source file exists")
	require.True(t, fileExists(testFs, filepath.Join(wantTitle, "index.md")), "File moved")
}

func TestMissingAttachment(t *testing.T) {
	src := "sample.md"
	dest := "new_folder"
	v := &bytes.Buffer{}
	log.SetOutput(v)

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", src), src)
	err := testFs.Mkdir(dest, os.ModePerm)
	failOn(t, err, "create directory")

	err = mv(testFs, src, dest)

	require.NoError(t, err)
	require.Contains(t, v.String(), "attachment is missing", "A warning should be logged")

	require.False(t, fileExists(testFs, src), "Source file still exists")
	require.True(t, fileExists(testFs, dest), "Destination file exists")
}

func TestFilenameCollision(t *testing.T) {
	file1 := "sample.md"
	file2 := "second.md"

	v := &bytes.Buffer{}
	log.SetOutput(v)

	src := "./*/index.md"
	dest := "folder"

	testFs := newTmpFS(t)
	copyFile(t, testFs, filepath.Join("testdata", file1), "folder1/index.md")
	copyFile(t, testFs, filepath.Join("testdata", file2), "folder2/index.md")
	copyFile(t, testFs, filepath.Join("testdata", file2), "folder3/index.md")
	err := testFs.Mkdir(dest, os.ModePerm)
	failOn(t, err, "create directory")

	err = mv(testFs, src, dest)
	require.NoError(t, err)

	require.Contains(t, v.String(), "same name", "A warning should be logged")

	require.False(t, dirExists(testFs, "folder1"), "First source directory exists")
	require.False(t, dirExists(testFs, "folder2"), "Second source directory exists")
	require.False(t, dirExists(testFs, "folder3"), "Third source directory exists")

	require.True(t, fileExists(testFs, filepath.Join(dest, "index.md")), "First file moved")
	require.True(t, fileExists(testFs, filepath.Join(dest, "index_1.md")), "Second file moved")
	require.True(t, fileExists(testFs, filepath.Join(dest, "index_2.md")), "Third file moved")
}

func newTmpFS(t *testing.T) afero.Fs {
	t.Helper()
	testFs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	t.Cleanup(func() {
		if t.Failed() {
			_ = printFilesystemState(testFs)
		}
	})
	return testFs
}

func printFilesystemState(testFs afero.Fs) error {
	err := afero.Walk(testFs, ".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to show filesystem state: %w", err)
	}
	return nil
}

func fileExists(fs afero.Fs, path string) bool {
	_, err := fs.Stat(path)
	return err == nil
}

func dirExists(fs afero.Fs, path string) bool {
	if f, err := fs.Stat(path); err == nil {
		return f.IsDir()
	}

	return false
}

func failOn(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatal(fmt.Errorf("%s: %w", message, err))
	}
}

func copyFile(t *testing.T, fs afero.Fs, src, dest string) {
	t.Helper()

	s, err := os.Open(src)
	failOn(t, err, "open source")

	if !dirExists(fs, filepath.Dir(dest)) {
		err = fs.MkdirAll(filepath.Dir(dest), os.ModePerm)
		failOn(t, err, "create directory tree")
	}
	d, err := fs.Create(dest)
	failOn(t, err, "create destination")

	defer func() {
		_, _ = s.Close(), d.Close()
	}()

	bb, err := io.Copy(d, s)
	failOn(t, err, "copy content")

	if bb == 0 {
		t.Fatal("no data was copied")
	}
}
