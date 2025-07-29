package oscheck

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse/managesw-mcp/internal/pkg/testenv"
)

const dnfPath = "bin/dnf"
const rpmPath = "bin/rpm"
const zypperPath = "bin/zypper"

func newTestEnv(t *testing.T) (env *testenv.TestEnv, path string) {
	env = testenv.New(t)
	path = env.GetPath("bin")
	env.MkdirAll("bin")
	return
}

func TestZypper(t *testing.T) {
	env, path := newTestEnv(t)
	defer env.RemoveAll()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", path)
	defer os.Setenv("PATH", oldPath)

	env.WriteFile(rpmPath, `#!/bin/sh
echo "rpm version 42"
`)
	err := os.Chmod(env.GetPath(rpmPath), 0755)
	assert.NoError(t, err)

	env.WriteFile(zypperPath, `#!/bin/sh
echo "zypper version 42"
`)
	err = os.Chmod(env.GetPath(zypperPath), 0755)
	assert.NoError(t, err)

	pkg := NewPkg(env.GetPath("/"))
	assert.Equal(t, "rpm", pkg.SysPackageInterface.PkgType())
}

func TestDnf(t *testing.T) {
	env, path := newTestEnv(t)
	defer env.RemoveAll()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", path)
	defer os.Setenv("PATH", oldPath)

	env.WriteFile(rpmPath, `#!/bin/sh
echo "rpm version 42"
`)
	err := os.Chmod(env.GetPath(rpmPath), 0755)
	assert.NoError(t, err)

	env.WriteFile(dnfPath, `#!/bin/sh
echo "dnf version 42"
`)
	err = os.Chmod(env.GetPath(dnfPath), 0755)
	assert.NoError(t, err)

	pkg := NewPkg(env.GetPath("/"))
	assert.Equal(t, "rpm", pkg.SysPackageInterface.PkgType())
}

func TestRpm(t *testing.T) {
	env, path := newTestEnv(t)
	defer env.RemoveAll()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", path)
	defer os.Setenv("PATH", oldPath)

	env.WriteFile(rpmPath, `#!/bin/sh
echo "rpm version 42"
`)
	err := os.Chmod(env.GetPath(rpmPath), 0755)
	assert.NoError(t, err)

	pkg := NewPkg(env.GetPath("/"))
	assert.Equal(t, "nopkg", pkg.SysPackageInterface.PkgType())
}

func TestNoPkg(t *testing.T) {
	env, path := newTestEnv(t)
	defer env.RemoveAll()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", path)
	defer os.Setenv("PATH", oldPath)

	pkg := NewPkg(env.GetPath("/"))
	assert.Equal(t, "nopkg", pkg.SysPackageInterface.PkgType())
}
