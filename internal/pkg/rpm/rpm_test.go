//go:build integration
// +build integration

package rpm

import (
	"path/filepath"
	"testing"

	"github.com/suse/managesw-mcp/internal/pkg/testenv"
	"github.com/stretchr/testify/assert"
)

func TestListInstalledPackagesSysCall(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Path to the RPM files
	rpmPath := "../../../test/rpmbuild/RPMS/x86_64/"

	// Import RPMs
	env.ImportRpm(filepath.Join(rpmPath, "base-1.0-1.x86_64.rpm"))
	env.ImportRpm(filepath.Join(rpmPath, "child-1.0-1.x86_64.rpm"))
	env.ImportRpm(filepath.Join(rpmPath, "grandchild-1.0-1.x86_64.rpm"))

	// Create a new RPM instance for testing
	rpm := NewRPMTest("rpm", Zypper, "zypper", env.GetPath(""))

	// List all installed packages
	pkgs, err := rpm.ListInstalledPackagesSysCall("")
	assert.NoError(t, err)
	assert.Len(t, pkgs, 3, "Expected 3 packages to be installed")

	// Check for a specific package
	basePkgs, err := rpm.ListInstalledPackagesSysCall("base")
	assert.NoError(t, err)
	assert.Len(t, basePkgs, 1, "Expected to find the 'base' package")
	assert.Equal(t, "base", basePkgs[0].Name)
	assert.Equal(t, "1.0", basePkgs[0].Version)
}
