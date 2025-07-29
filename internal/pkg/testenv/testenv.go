// Package testenv provides functions and data structures for
// constructing and manipulating a temporary environment for
// use during automated testing.
//
// The testenv package should only be used in tests.
package testenv

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestEnv struct {
	t       *testing.T
	BaseDir string
}

const Rpmdir = "rpm"
const Repodir = "repo"
const Sysconfdir = "etc"

// New creates a test environment in a temporary directory.
//
// Caller is responsible to delete env.BaseDir by calling
// env.RemoveAll.
//
// Asserts no errors occur.
func New(t *testing.T) (env *TestEnv) {
	env = new(TestEnv)
	env.t = t
	env.init()
	return env
}

// init creates a temporary directory and initializes an RPM database.
//
// Asserts no errors occur.
func (env *TestEnv) init() {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "managesw-mcp-test-*")
	assert.NoError(env.t, err)
	env.BaseDir = tmpDir

	// Create directory for RPM database
	env.MkdirAll("var/lib/rpm")

	// Initialize RPM database
	cmd := exec.Command("rpm", "--root", env.GetPath("/"), "--initdb")
	output, err := cmd.CombinedOutput()
	assert.NoError(env.t, err, string(output))

	env.MkdirAll(Rpmdir)
	env.MkdirAll(Repodir)
	env.MkdirAll(Sysconfdir)
}

// GetPath returns the absolute path name for fileName specified
// relative to the test environment.
func (env *TestEnv) GetPath(fileName string) string {
	return path.Join(env.BaseDir, fileName)
}

// MkdirAll creates dirName and any intermediate directories relative
// to the test environment.
//
// Asserts no errors occur.
func (env *TestEnv) MkdirAll(dirName string) {
	err := os.MkdirAll(env.GetPath(dirName), 0755)
	assert.NoError(env.t, err)
}

// WriteFile writes content to fileName, creating any necessary
// intermediate directories relative to the test environment.
//
// Asserts no errors occur.
func (env *TestEnv) WriteFile(fileName string, content string) {
	dirName := filepath.Dir(fileName)
	env.MkdirAll(dirName)

	f, err := os.Create(env.GetPath(fileName))
	assert.NoError(env.t, err)
	defer f.Close()
	_, err = f.WriteString(content)
	assert.NoError(env.t, err)
}

// ImportFile writes the contents of inputFileName to fileName,
// creating any necessary intermediate directories relative to the
// test environment.
func (env *TestEnv) ImportFile(fileName string, inputFileName string) {
	buffer, err := os.ReadFile(inputFileName)
	assert.NoError(env.t, err)
	env.WriteFile(fileName, string(buffer))
}

func (env *TestEnv) ImportDir(dirName string, inputDirName string) {
	env.MkdirAll(path.Dir(dirName))
	cmd := exec.Command("cp", "--recursive", inputDirName, env.GetPath(dirName))
	output, err := cmd.CombinedOutput()
	assert.NoError(env.t, err, string(output))
}

// ImportRpm installs an RPM into the test environment's RPM database
// and copies it to the rpm directory.
func (env *TestEnv) ImportRpm(rpmPath string) {
	// Install RPM
	cmd := exec.Command("rpm", "-i", "--nodeps", "--root", env.GetPath("/"), rpmPath)
	output, err := cmd.CombinedOutput()
	assert.NoError(env.t, err, string(output))

	// Copy RPM to rpm directory
	_, file := filepath.Split(rpmPath)
	env.ImportFile(filepath.Join(Rpmdir, file), rpmPath)
}

// RemoveAll deletes the temporary directory, and all its contents,
// for the test environment.
//
// Asserts no errors occur.
func (env *TestEnv) RemoveAll() {
	err := os.RemoveAll(env.BaseDir)
	assert.NoError(env.t, err)
}
