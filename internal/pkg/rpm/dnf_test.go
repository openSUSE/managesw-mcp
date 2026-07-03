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

func TestDnfRepoManagement(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	binDir := env.GetPath("bin")
	env.MkdirAll("bin")

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Mock dnf to simulate repo management
	dnfMock := `#!/bin/sh
STATE_DIR="` + env.GetPath("") + `"
if [ "$1" = "repo" ] && [ "$2" = "list" ]; then
	if [ -f "$STATE_DIR/repo_added" ]; then
		echo "Repo-id: test-repo"
		if [ -f "$STATE_DIR/repo_disabled" ]; then
			echo "Repo-status: disabled"
		else
			echo "Repo-status: enabled"
		fi
		echo "Repo-baseurl: http://example.com/dnf"
	fi
elif [ "$1" = "repo" ] && [ "$2" = "modify" ]; then
	if [ ! -f "$STATE_DIR/repo_added" ]; then
		exit 1
	fi
	has_disable=0
	for arg in "$@"; do
		if [ "$arg" = "--disable" ]; then
			has_disable=1
		fi
	done
	if [ $has_disable -eq 1 ]; then
		touch "$STATE_DIR/repo_disabled"
	else
		rm -f "$STATE_DIR/repo_disabled"
	fi
elif [ "$1" = "repo" ] && [ "$2" = "remove" ]; then
	rm -f "$STATE_DIR/repo_added"
	rm -f "$STATE_DIR/repo_disabled"
elif [ "$1" = "config-manager" ] && [ "$2" = "--add-repo" ]; then
	touch "$STATE_DIR/repo_added"
fi
`
	env.WriteFile("bin/dnf", dnfMock)
	err := os.Chmod(env.GetPath("bin/dnf"), 0755)
	require.NoError(t, err)

	rpm := NewRPMTest("rpm", Dnf, env.GetPath("bin/dnf"), "")

	// 1. List repos initially (should be empty)
	repos, err := rpm.ListReposSysCall("")
	require.NoError(t, err)
	assert.Empty(t, repos)

	// 2. Add repository (ModifyRepo with URL when it doesn't exist)
	addParams := syspackage.ModifyRepoParams{
		Name: "test-repo",
		Url:  "http://example.com/dnf",
	}
	repo, err := rpm.ModifyRepoSysCall(addParams)
	require.NoError(t, err)
	assert.Equal(t, "test-repo", repo["Repo-id"])
	assert.Equal(t, "enabled", repo["Repo-status"])

	// 3. Verify it is listed
	repos, err = rpm.ListReposSysCall("")
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "test-repo", repos[0]["Repo-id"])

	// 4. Disable repository
	disableParams := syspackage.ModifyRepoParams{
		Name:    "test-repo",
		Disable: true,
	}
	repo, err = rpm.ModifyRepoSysCall(disableParams)
	require.NoError(t, err)
	assert.Equal(t, "disabled", repo["Repo-status"])

	// 5. Enable repository
	enableParams := syspackage.ModifyRepoParams{
		Name:    "test-repo",
		Disable: false,
	}
	repo, err = rpm.ModifyRepoSysCall(enableParams)
	require.NoError(t, err)
	assert.Equal(t, "enabled", repo["Repo-status"])

	// 6. Refresh repository
	err = rpm.RefreshReposSysCall("test-repo")
	require.NoError(t, err)

	// 7. Remove repository
	removeParams := syspackage.ModifyRepoParams{
		Name:        "test-repo",
		RemoveRepos: true,
	}
	_, err = rpm.ModifyRepoSysCall(removeParams)
	require.NoError(t, err)

	// 8. Verify repository is removed
	repos, err = rpm.ListReposSysCall("")
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestDnfInstallPackage(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	binDir := env.GetPath("bin")
	env.MkdirAll("bin")

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Mock dnf to simulate install commands and verify weak deps configuration
	dnfMock := `#!/bin/sh
# Log all args to a file to verify correct flags are passed
echo "$@" >> "` + env.GetPath("dnf_args.log") + `"
echo "Installing package test-pkg..."
`
	env.WriteFile("bin/dnf", dnfMock)
	err := os.Chmod(env.GetPath("bin/dnf"), 0755)
	require.NoError(t, err)

	rpm := NewRPMTest("rpm", Dnf, env.GetPath("bin/dnf"), "")

	// Case 1: Default install (should install weak deps, meaning NoRecommends is false by default)
	output, err := rpm.InstallPackageSysCall(nil, nil, syspackage.InstallPackageParams{
		Name: "test-pkg",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "Installing package test-pkg...")

	// Case 2: Install with NoRecommends set to true
	_, err = rpm.InstallPackageSysCall(nil, nil, syspackage.InstallPackageParams{
		Name:         "test-pkg-no-rec",
		NoRecommends: true,
	})
	require.NoError(t, err)

	// Verify the arguments passed to dnf mock
	argsLog, err := os.ReadFile(env.GetPath("dnf_args.log"))
	require.NoError(t, err)
	argsStr := string(argsLog)
	
	// Default: NoRecommends: false -> --setopt=install_weak_deps=True
	assert.Contains(t, argsStr, "install -y --setopt=install_weak_deps=True test-pkg")
	// NoRecommends: true -> --setopt=install_weak_deps=False
	assert.Contains(t, argsStr, "install -y --setopt=install_weak_deps=False test-pkg-no-rec")
}

func TestZypperInstallPackage(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	binDir := env.GetPath("bin")
	env.MkdirAll("bin")

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Mock zypper to simulate install commands and verify recommends configuration
	zypperMock := `#!/bin/sh
# Log all args to a file to verify correct flags are passed
echo "$@" >> "` + env.GetPath("zypper_args.log") + `"
echo "Installing package test-pkg..."
`
	env.WriteFile("bin/zypper", zypperMock)
	err := os.Chmod(env.GetPath("bin/zypper"), 0755)
	require.NoError(t, err)

	rpm := NewRPMTest("rpm", Zypper, env.GetPath("bin/zypper"), "")

	// Case 1: Default install (should install recommended packages by default, so --no-recommends is NOT passed)
	output, err := rpm.InstallPackageSysCall(nil, nil, syspackage.InstallPackageParams{
		Name: "test-pkg",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "Installing package test-pkg...")

	// Case 2: Install with NoRecommends set to true
	_, err = rpm.InstallPackageSysCall(nil, nil, syspackage.InstallPackageParams{
		Name:         "other-pkg",
		NoRecommends: true,
	})
	require.NoError(t, err)

	// Verify the arguments passed to zypper mock
	argsLog, err := os.ReadFile(env.GetPath("zypper_args.log"))
	require.NoError(t, err)
	argsStr := string(argsLog)
	
	// Default: NoRecommends is false, so --no-recommends should NOT be present for test-pkg
	assert.Contains(t, argsStr, "install test-pkg")
	assert.NotContains(t, argsStr, "--no-recommends test-pkg")

	// NoRecommends: true -> --no-recommends should be present
	assert.Contains(t, argsStr, "install --no-recommends other-pkg")
}
