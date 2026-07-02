//go:build integration
// +build integration

package rpm

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
	"github.com/suse/managesw-mcp/internal/pkg/testenv"
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
	pkgs, err := rpm.ListInstalledPackagesSysCall(syspackage.ListPackageParams{})
	assert.NoError(t, err)
	assert.Len(t, pkgs, 3, "Expected 3 packages to be installed")

	// Check for a specific package with details
	basePkgs, err := rpm.ListInstalledPackagesSysCall(syspackage.ListPackageParams{
		Name:        "base",
		Filelist:    true,
		Description: true,
		Relations:   []string{"requires", "provides", "conflicts", "suggests"},
		Changelog:   3,
	})
	assert.NoError(t, err)
	assert.Len(t, basePkgs, 1, "Expected to find the 'base' package")
	assert.Equal(t, "base", basePkgs[0].Name)
	assert.Equal(t, "1.0", basePkgs[0].Version)
	assert.NotEmpty(t, basePkgs[0].FileList, "Expected base package to have file list")
	assert.NotEmpty(t, basePkgs[0].Description, "Expected base package to have description")
	assert.Contains(t, basePkgs[0].Relations, "requires", "Expected base package to have requires relations")
	assert.Contains(t, basePkgs[0].Relations, "provides", "Expected base package to have provides relations")
	assert.Contains(t, basePkgs[0].Relations, "conflicts", "Expected base package to have conflicts relations")
	assert.Contains(t, basePkgs[0].Relations, "suggests", "Expected base package to have suggests relations")
}
