package dpkg

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

type DPKG struct {
	dpkgbin   string
	dpkgquery string
}

func New(dpkgbin string, dpkgquery string) DPKG {
	return DPKG{
		dpkgbin:   dpkgbin,
		dpkgquery: dpkgquery,
	}
}

func (dpkg DPKG) ListInstalledPackagesSysCall(name string) (lst []syspackage.SysPackageInfo, err error) {
	pkgList, err := exec.Command(dpkg.dpkgquery, "-W", "-f", "'${binary:Package},${Version},${Installed-Size}\n'", name).CombinedOutput()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(pkgList))
	for scanner.Scan() {
		splitLine := strings.Split(scanner.Text(), ",")
		if len(splitLine) != 3 {
			continue
		}
		size, _ := strconv.Atoi(splitLine[2])
		lst = append(lst, syspackage.SysPackageInfo{
			Name:    splitLine[0],
			Version: splitLine[1],
			Size:    uint64(size),
		})
	}

	return lst, nil
}
