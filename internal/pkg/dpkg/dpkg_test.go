package dpkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
	"github.com/suse/managesw-mcp/internal/pkg/testenv"
)

func TestDpkgSearchPackage(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	binDir := env.GetPath("bin")
	env.MkdirAll("bin")

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", oldPath)

	// Mock dpkg-query to simulate listing installed packages
	dpkgQueryMock := `#!/bin/sh
if [ "$1" = "-W" ] && [ "$2" = "-f" ]; then
    # We are querying installed packages.
    # Return one matching package: "test-pkg"
    echo "test-pkg,1.2.3-1,1024"
fi
`
	env.WriteFile("bin/dpkg-query", dpkgQueryMock)
	err := os.Chmod(env.GetPath("bin/dpkg-query"), 0755)
	require.NoError(t, err)

	// Mock apt-cache to simulate search and madison
	aptCacheMock := `#!/bin/sh
if [ "$1" = "search" ]; then
    # Return list of matching packages
    echo "test-pkg - A test package description"
    echo "other-pkg - Another package"
elif [ "$1" = "madison" ]; then
    # Return madison structured output
    echo " test-pkg | 1.2.3-1 | http://deb.debian.org/debian bookworm/main amd64 Packages"
    echo " other-pkg | 2.0.0-1 | http://deb.debian.org/debian bookworm/main amd64 Packages"
fi
`
	env.WriteFile("bin/apt-cache", aptCacheMock)
	err = os.Chmod(env.GetPath("bin/apt-cache"), 0755)
	require.NoError(t, err)

	// Create DPKG instance
	d := New("dpkg", env.GetPath("bin/dpkg-query"), env.GetPath("bin/apt-cache"))

	// Search for packages matching "test"
	pkgsAny, err := d.SearchPackageSysCall(syspackage.SearchPackageParams{Name: "test"})
	require.NoError(t, err)

	pkgs, ok := pkgsAny.(map[string]map[string][]syspackage.SearchedPackage)
	require.True(t, ok, "Expected search output to be map[string]map[string][]syspackage.SearchedPackage")

	// Verify available packages from repository
	assert.Contains(t, pkgs, "http://deb.debian.org/debian")
	assert.Contains(t, pkgs["http://deb.debian.org/debian"], "amd64")
	require.Len(t, pkgs["http://deb.debian.org/debian"]["amd64"], 2)

	// Verify statuses
	var foundTestPkg, foundOtherPkg bool
	for _, pkg := range pkgs["http://deb.debian.org/debian"]["amd64"] {
		if pkg.Name == "test-pkg" {
			foundTestPkg = true
			assert.Equal(t, "1.2.3-1", pkg.Version)
			assert.Equal(t, "v", pkg.Status)
		} else if pkg.Name == "other-pkg" {
			foundOtherPkg = true
			assert.Equal(t, "2.0.0-1", pkg.Version)
			assert.Equal(t, "v", pkg.Status)
		}
	}
	assert.True(t, foundTestPkg)
	assert.True(t, foundOtherPkg)

	// Verify installed package under "System" repo
	assert.Contains(t, pkgs, "System")
	assert.Contains(t, pkgs["System"], "unknown")
	require.Len(t, pkgs["System"]["unknown"], 1)
	assert.Equal(t, "test-pkg", pkgs["System"]["unknown"][0].Name)
	assert.Equal(t, "1.2.3-1", pkgs["System"]["unknown"][0].Version)
	assert.Equal(t, "i", pkgs["System"]["unknown"][0].Status)
}
