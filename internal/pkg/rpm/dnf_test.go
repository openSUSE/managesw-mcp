package rpm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
	"github.com/suse/managesw-mcp/internal/pkg/testenv"
)

func TestDnfSearchPackage(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	binDir := env.GetPath("bin")
	env.MkdirAll("bin")

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", oldPath)

	// Mock dnf to simulate repoquery
	dnfMock := `#!/bin/sh
is_repoquery=0
for arg in "$@"; do
    if [ "$arg" = "repoquery" ]; then
        is_repoquery=1
    fi
done

if [ $is_repoquery -eq 1 ]; then
    # Return simulated repoquery tab-separated output: name, repoid, arch, version
    echo "test-pkg	@System	x86_64	1.2.3-1"
    echo "test-pkg	fedora	x86_64	1.2.3-1"
    echo "other-pkg	fedora	noarch	2.0.0-1"
fi
`
	env.WriteFile("bin/dnf", dnfMock)
	err := os.Chmod(env.GetPath("bin/dnf"), 0755)
	require.NoError(t, err)

	// Create RPM instance with Df manager
	rpm := NewRPMTest("rpm", Dnf, env.GetPath("bin/dnf"), "")

	// Search for packages matching "test"
	pkgsAny, err := rpm.SearchPackageSysCall(syspackage.SearchPackageParams{Name: "test"})
	require.NoError(t, err)

	pkgs, ok := pkgsAny.(map[string]map[string][]syspackage.SearchedPackage)
	require.True(t, ok, "Expected search output to be map[string]map[string][]syspackage.SearchedPackage")

	// Verify packages under "System" repo (originally @System)
	assert.Contains(t, pkgs, "System")
	assert.Contains(t, pkgs["System"], "x86_64")
	require.Len(t, pkgs["System"]["x86_64"], 1)
	assert.Equal(t, "test-pkg", pkgs["System"]["x86_64"][0].Name)
	assert.Equal(t, "1.2.3-1", pkgs["System"]["x86_64"][0].Version)
	assert.Equal(t, "i", pkgs["System"]["x86_64"][0].Status)

	// Verify packages under "fedora" repo
	assert.Contains(t, pkgs, "fedora")
	assert.Contains(t, pkgs["fedora"], "x86_64")
	assert.Contains(t, pkgs["fedora"], "noarch")

	require.Len(t, pkgs["fedora"]["x86_64"], 1)
	assert.Equal(t, "test-pkg", pkgs["fedora"]["x86_64"][0].Name)
	assert.Equal(t, "1.2.3-1", pkgs["fedora"]["x86_64"][0].Version)
	assert.Equal(t, "v", pkgs["fedora"]["x86_64"][0].Status)

	require.Len(t, pkgs["fedora"]["noarch"], 1)
	assert.Equal(t, "other-pkg", pkgs["fedora"]["noarch"][0].Name)
	assert.Equal(t, "2.0.0-1", pkgs["fedora"]["noarch"][0].Version)
	assert.Equal(t, "v", pkgs["fedora"]["noarch"][0].Status)
}
