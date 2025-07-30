package syspackage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse/managesw-mcp/internal/pkg/nopkgs"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
	"github.com/suse/managesw-mcp/internal/pkg/testenv"
)

func TestSysPackageNoPkg(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	var sysPkg syspackage.SysPackage = syspackage.SysPackage{
		SysPackageInterface: &nopkgs.NoPkg{},
	}

	assert.Equal(t, "nopkg", sysPkg.PkgType(), "Should be nopkg")
}
