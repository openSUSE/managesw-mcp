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

type mockSysPackage struct {
	nopkgs.NoPkg
}

func (m mockSysPackage) ListReposSysCall(name string) ([]map[string]any, error) {
	return []map[string]any{
		{"alias": "repo1", "name": "Repo 1"},
		{"Repo-id": "repo2", "name": "Repo 2"},
		{"id": "repo3", "name": "Repo 3"},
	}, nil
}

func TestCreateSearchPackageSchema(t *testing.T) {
	// Case 1: NoPkg (where ListReposSysCall fails or is not implemented)
	sysPkgNoPkg := syspackage.SysPackage{
		SysPackageInterface: &nopkgs.NoPkg{},
	}
	schema, err := sysPkgNoPkg.CreateSearchPackageSchema()
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Contains(t, schema.Properties, "repos")
	if schema.Properties["repos"].Items != nil {
		assert.Empty(t, schema.Properties["repos"].Items.Enum)
	}

	// Case 2: Mock with repos
	sysPkgMock := syspackage.SysPackage{
		SysPackageInterface: &mockSysPackage{},
	}
	schemaMock, err := sysPkgMock.CreateSearchPackageSchema()
	assert.NoError(t, err)
	assert.NotNil(t, schemaMock)
	assert.Contains(t, schemaMock.Properties, "repos")
	assert.NotNil(t, schemaMock.Properties["repos"].Items)
	assert.Equal(t, []any{"repo1", "repo2", "repo3"}, schemaMock.Properties["repos"].Items.Enum)
}

func TestCreateInstallPackageSchema(t *testing.T) {
	// Case 1: NoPkg
	sysPkgNoPkg := syspackage.SysPackage{
		SysPackageInterface: &nopkgs.NoPkg{},
	}
	schema, err := sysPkgNoPkg.CreateInstallPackageSchema()
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Contains(t, schema.Properties, "repo")
	assert.Empty(t, schema.Properties["repo"].Enum)

	// Case 2: Mock with repos
	sysPkgMock := syspackage.SysPackage{
		SysPackageInterface: &mockSysPackage{},
	}
	schemaMock, err := sysPkgMock.CreateInstallPackageSchema()
	assert.NoError(t, err)
	assert.NotNil(t, schemaMock)
	assert.Contains(t, schemaMock.Properties, "repo")
	assert.Equal(t, []any{"repo1", "repo2", "repo3"}, schemaMock.Properties["repo"].Enum)
}
