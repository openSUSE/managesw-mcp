package rpm

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

type RPMType int

const (
	Zypper = iota
	Dnf
)

type PkgMgr struct {
	mgrtype RPMType
	mgrpath string
}

type RPM struct {
	rpmpath string
	mgr     PkgMgr
}

func NewRPM(path string, systype RPMType, mgrpath string) RPM {
	return RPM{
		rpmpath: path,
		mgr: PkgMgr{
			mgrtype: systype,
			mgrpath: mgrpath,
		},
	}
}

// ListInstalledPackagesSysCall lists the installed packages given by their name pattern.
func (rpm RPM) ListInstalledPackagesSysCall(name string) ([]syspackage.SysPackageInfo, error) {
	// The query format doesn't need shell quoting since exec.Command passes arguments directly.
	qf := `%{NAME},%{VERSION},%{SIZE}\n`
	args := []string{"-qa", "--qf", qf}
	if name != "" {
		args = append(args, name)
	}
	cmd := exec.Command(rpm.rpmpath, args...)
	pkgList, err := cmd.CombinedOutput()

	// rpm exits with 1 if no packages are found. This is not an error for us.
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// No packages found, return empty list
			return []syspackage.SysPackageInfo{}, nil
		}
		return nil, fmt.Errorf("rpm command failed: %w, output: %s", err, string(pkgList))
	}

	var lst []syspackage.SysPackageInfo
	scanner := bufio.NewScanner(bytes.NewReader(pkgList))
	for scanner.Scan() {
		line := scanner.Text()
		// The output might have leading/trailing single quotes from the old command format, let's be robust.
		line = strings.Trim(line, "'")
		splitLine := strings.Split(line, ",")
		if len(splitLine) != 3 {
			continue
		}
		size, err := strconv.ParseUint(splitLine[2], 10, 64)
		if err != nil {
			// If size is not a valid number, we can either skip the package or set size to 0.
			// Setting to 0 seems like a reasonable default.
			size = 0
		}
		lst = append(lst, syspackage.SysPackageInfo{
			Name:    splitLine[0],
			Version: splitLine[1],
			Size:    size,
		})
	}

	return lst, nil
}

// QueryPackageSyscall queries package information.
func (rpm RPM) QueryPackageSysCall(name string, mode syspackage.QueryMode, lines int) (result map[string]any, err error) {
	var cmdArgs []string
	var resultKey string

	switch mode {
	case syspackage.Info:
		cmdArgs = []string{"-qi", name}
		resultKey = "info"
	case syspackage.Requires:
		cmdArgs = []string{"-q", "--requires", name}
		resultKey = "requires"
	case syspackage.Recommends:
		cmdArgs = []string{"-q", "--recommends", name}
		resultKey = "recommends"
	case syspackage.Obsoletes:
		cmdArgs = []string{"-q", "--obsoletes", name}
		resultKey = "obsoletes"
	case syspackage.Changelog:
		cmdArgs = []string{"-q", "--changelog", name}
		resultKey = "changelog"
	default:
		return nil, fmt.Errorf("unsupported query mode: %v", mode)
	}

	output, err := exec.Command(rpm.rpmpath, cmdArgs...).CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// Package not found
			return nil, fmt.Errorf("package not found: %s", name)
		}
		return nil, fmt.Errorf("failed to query package '%s': %w. Output: %s", name, err, string(output))
	}
	result = make(map[string]any)
	if mode == syspackage.Info {
		// For info, parse the key-value output.
		scanner := bufio.NewScanner(bytes.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result[key] = value
			}
		}
	} else {
		// For other modes, return the full output under a single key.
		splittedLines := strings.Split(string(output), "\n")
		if lines > 0 && len(splittedLines) > lines {
			result[resultKey] = splittedLines[:lines]
		} else {
			result[resultKey] = splittedLines
		}
	}

	return result, nil
}

func (rpm RPM) ListReposSysCall() ([]map[string]any, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return listReposZypper()
	case Dnf:
		return listReposDnf()
	default:
		return nil, fmt.Errorf("No rpm package manager installed")
	}
}
