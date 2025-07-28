package nopkgs

import (
	"fmt"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

type NoPkg struct{}

func (n NoPkg) ListInstalledPackagesSysCall(name string) ([]syspackage.SysPackageInfo, error) {
	return []syspackage.SysPackageInfo{}, fmt.Errorf("No package manager found")
}
func (n NoPkg) QueryPackageSysCall(name string, mode syspackage.QueryMode, lines int) (ret map[string]any, err error) {
	return ret, fmt.Errorf("No package manager found")
}
func (n NoPkg) ListReposSysCall(name string) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n NoPkg) ModifyRepoSysCall(params syspackage.ModifyRepoParams) (map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n NoPkg) ListPatchesSysCall(params syspackage.ListPatchesParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n NoPkg) InstallPatchesSysCall(params syspackage.InstallPatchesParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n NoPkg) RefreshReposSysCall(name string) error {
	return fmt.Errorf("not implemented")
}

func (n NoPkg) SearchPackage(params syspackage.SearchPackageParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}
