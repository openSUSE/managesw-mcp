package nopkgs

import (
	"fmt"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

type NoPkg struct{}

func (n NoPkg) ListInstalledPackagesSysCall(name string) ([]syspackage.SysPackageInfo, error) {
	return []syspackage.SysPackageInfo{}, fmt.Errorf("No package manager found")
}
func (n NoPkg) QueryPackageSyscall(name string, mode syspackage.QueryMode, lines int) (ret map[string]any, err error) {
	return ret, fmt.Errorf("No package manager found")
}
