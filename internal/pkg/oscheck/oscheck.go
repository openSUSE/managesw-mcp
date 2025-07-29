package oscheck

import (
	"os/exec"

	"github.com/suse/managesw-mcp/internal/pkg/dpkg"
	"github.com/suse/managesw-mcp/internal/pkg/nopkgs"
	"github.com/suse/managesw-mcp/internal/pkg/rpm"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

func NewPkg(root string) syspackage.SysPackage {
	if rpmpath, err := exec.LookPath("rpm"); err == nil {
		args := []string{}
		if root != "" {
			args = append(args, "--root", root)
		}
		args = append(args, "-q", "rpm")
		cmd := exec.Command(rpmpath, args...)
		if err := cmd.Run(); err == nil {
			if zypperPath, err := exec.LookPath("zypper"); err == nil {
				return syspackage.SysPackage{rpm.NewRPM(rpmpath, rpm.Zypper, zypperPath, root)}
			}
			if dnfPath, err := exec.LookPath("dnf"); err == nil {
				return syspackage.SysPackage{rpm.NewRPM(rpmpath, rpm.Dnf, dnfPath, root)}
			}
		}
	}
	dpkgpath, err := exec.LookPath("dpkg")
	if err == nil {
		dpkgquery, err := exec.LookPath("dpkg-query")
		if err != nil {
			goto nodpkg
		}
		args := []string{}
		if root != "" {
			args = append(args, "--root", root)
		}
		args = append(args, "-s", "dpkg")
		cmd := exec.Command(dpkgquery, args...)
		dpkgCmdOut, err := cmd.Output()
		if err == nil && len(dpkgCmdOut) > 0 {
			return syspackage.SysPackage{dpkg.New(dpkgpath, dpkgquery)}
		}
	}
nodpkg:
	return syspackage.SysPackage{nopkgs.NoPkg{}}

}
