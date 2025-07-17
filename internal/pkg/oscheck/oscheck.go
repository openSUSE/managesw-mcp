package oscheck

import (
	"os/exec"

	"github.com/suse/managesw-mcp/internal/pkg/dpkg"
	"github.com/suse/managesw-mcp/internal/pkg/nopkgs"
	"github.com/suse/managesw-mcp/internal/pkg/rpm"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

func NewPkg() syspackage.SysPackage {
	rpmpath, err := exec.LookPath("rpm")
	if err == nil {
		rpmCmdOut, err := exec.Command(rpmpath, "-q", "rpm").Output()
		if err == nil && len(rpmCmdOut) > 0 {
			return syspackage.SysPackage{rpm.New(rpmpath)}
		}
	}
	dpkgpath, err := exec.LookPath("dpkg")
	if err == nil {
		dpkgquery, err := exec.LookPath("dpkg-query")
		if err != nil {
			goto nodpkg
		}
		dpkgCmdOut, err := exec.Command(dpkgquery, "-s", "dpkg").Output()
		if err != nil && len(dpkgCmdOut) > 0 {
			return syspackage.SysPackage{dpkg.New(dpkgpath, dpkgquery)}
		}
	}
nodpkg:
	return syspackage.SysPackage{nopkgs.NoPkg{}}

}
