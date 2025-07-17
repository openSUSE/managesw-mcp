package rpm

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

type RPM struct {
	rpmpath string
}

func New(path string) RPM {
	return RPM{
		rpmpath: path,
	}
}

func (rpm RPM) ListInstalledPackagesSysCall(name string) (lst []syspackage.SysPackageInfo, err error) {
	pkgList, err := exec.Command(rpm.rpmpath, "-qa", "--qf", "'%{NAME},%{VERSION},%{SIZE}'\n", name).CombinedOutput()
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
