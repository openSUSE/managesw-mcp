// Package testenv provides functions and data structures for
// constructing and manipulating a temporary environment for
// use during automated testing.
//
// The testenv package should only be used in tests.
package testenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	env := New(t)
	defer env.RemoveAll()

	assert.NotNil(t, env)
	assert.NotEmpty(t, env.BaseDir)
	_, err := os.Stat(env.BaseDir)
	assert.NoError(t, err)

	// Check for rpm db
	rpmDbDir := filepath.Join(env.BaseDir, "var/lib/rpm")
	_, err = os.Stat(rpmDbDir)
	assert.NoError(t, err)
}

func TestGetPath(t *testing.T) {
	env := New(t)
	defer env.RemoveAll()

	path := env.GetPath("foo/bar")
	assert.Equal(t, filepath.Join(env.BaseDir, "foo/bar"), path)
}

func TestMkdirAll(t *testing.T) {
	env := New(t)
	defer env.RemoveAll()

	path := "foo/bar"
	env.MkdirAll(path)

	_, err := os.Stat(env.GetPath(path))
	assert.NoError(t, err)
}

func TestWriteFile(t *testing.T) {
	env := New(t)
	defer env.RemoveAll()

	path := "foo/bar.txt"
	content := "hello world"
	env.WriteFile(path, content)

	readContent, err := os.ReadFile(env.GetPath(path))
	assert.NoError(t, err)
	assert.Equal(t, content, string(readContent))
}

func TestImportFile(t *testing.T) {
	env := New(t)
	defer env.RemoveAll()

	// Create a temporary file to import
	tmpFile, err := os.CreateTemp("", "import-test-")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "import content"
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	importPath := "imported/file.txt"
	env.ImportFile(importPath, tmpFile.Name())

	readContent, err := os.ReadFile(env.GetPath(importPath))
	assert.NoError(t, err)
	assert.Equal(t, content, string(readContent))
}

func TestImportDir(t *testing.T) {
	env := New(t)
	defer env.RemoveAll()

	// Create a temporary directory to import
	tmpDir, err := os.MkdirTemp("", "import-dir-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a file in the temporary directory
	tmpFile, err := os.Create(filepath.Join(tmpDir, "file.txt"))
	assert.NoError(t, err)
	_, err = tmpFile.WriteString("hello")
	assert.NoError(t, err)
	tmpFile.Close()

	importPath := "imported_dir"
	env.ImportDir(importPath, tmpDir)

	_, err = os.Stat(env.GetPath(filepath.Join(importPath, "file.txt")))
	assert.NoError(t, err)
}

func TestRemoveAll(t *testing.T) {
	env := New(t)
	baseDir := env.BaseDir
	env.RemoveAll()

	_, err := os.Stat(baseDir)
	assert.True(t, os.IsNotExist(err))
}