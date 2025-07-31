//go:build integration
// +build integration

package rpm

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
	"github.com/suse/managesw-mcp/internal/pkg/testenv"
)

func TestRefreshReposSearch(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Create a local repo
	repoPath := env.GetPath("my-local-repo")
	env.MkdirAll("my-local-repo")

	repoContent := `
[my-local-repo]
name=My Local Repo
enabled=1
autorefresh=1
baseurl=dir://` + repoPath + `
type=rpm-md
gpgcheck=0
`
	env.WriteFile("etc/zypp/repos.d/my-local-repo.repo", repoContent)

	// Create a new RPM instance for testing with Zypper
	rpm := NewRPMTest("rpm", Zypper, "zypper", env.GetPath(""))

	rpmArch, err := exec.Command("rpm", "--eval", "%{_arch}").Output()
	require.NoError(t, err, "Failed to get RPM architecture")
	arch := strings.TrimSpace(string(rpmArch))
	baseRpmPath := "../../../test/rpmbuild/RPMS/" + arch + "/base-1.0-1." + arch + ".rpm"
	childRpmPath := "../../../test/rpmbuild/RPMS/" + arch + "/child-1.0-1." + arch + ".rpm"

	env.ImportFile(filepath.Join("my-local-repo", "base-1.0-1."+arch+".rpm"), baseRpmPath)

	// Refresh repos
	err = rpm.RefreshReposSysCall("my-local-repo")
	require.NoError(t, err)

	// List repos and check if it is correctly added
	repos, err := rpm.ListReposSysCall("")
	require.NoError(t, err)
	require.Len(t, repos, 1, "Expected to find 1 repo")
	assert.Equal(t, "my-local-repo", repos[0]["alias"])
	assert.Equal(t, "My Local Repo", repos[0]["name"])
	assert.Equal(t, "1", repos[0]["enabled"])
	assert.Equal(t, "1", repos[0]["autorefresh"])

	// Search for base package
	pkgs, err := rpm.SearchPackageSysCall(syspackage.SearchPackageParams{Name: "base"})
	require.NoError(t, err)
	require.Len(t, pkgs, 1, "Expected to find 1 package")
	assert.Equal(t, "base", pkgs[0]["name"])

	// Add child RPM and update repo
	env.ImportFile(filepath.Join("my-local-repo", "child-1.0-1."+arch+".rpm"), childRpmPath)

	// Refresh repos again
	err = rpm.RefreshReposSysCall("my-local-repo")
	require.NoError(t, err)

	// Search for child package
	pkgs, err = rpm.SearchPackageSysCall(syspackage.SearchPackageParams{Name: "child"})
	require.Len(t, pkgs, 1, "Expected to find 1 package")
	assert.Equal(t, "child", pkgs[0]["name"])
}
