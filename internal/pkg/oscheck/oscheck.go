package oscheck

import (
	"os/exec"

	"github.com/suse/managesw-mcp/internal/pkg/dpkg"
	"github.com/suse/managesw-mcp/internal/pkg/nopkgs"
	"github.com/suse/managesw-mcp/internal/pkg/rpm"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

func NewPkg() syspackage.SysPackage {
	if rpmpath, err := exec.LookPath("rpm"); err == nil {
		if err := exec.Command(rpmpath, "-q", "rpm").Run(); err == nil {
			if zypperPath, err := exec.LookPath("zypper"); err == nil {
				return syspackage.SysPackage{rpm.NewRPM(rpmpath, rpm.Zypper, zypperPath)}
			}
			if dnfPath, err := exec.LookPath("dnf"); err == nil {
				return syspackage.SysPackage{rpm.NewRPM(rpmpath, rpm.Dnf, dnfPath)}
			}
		}
	}
	dpkgpath, err := exec.LookPath("dpkg")
	if err == nil {
		dpkgquery, err := exec.LookPath("dpkg-query")
		if err != nil {
			goto nodpkg
		}
		dpkgCmdOut, err := exec.Command(dpkgquery, "-s", "dpkg").Output()
		if err == nil && len(dpkgCmdOut) > 0 {
			return syspackage.SysPackage{dpkg.New(dpkgpath, dpkgquery)}
		}
	}
nodpkg:
	return syspackage.SysPackage{nopkgs.NoPkg{}}

}
