package dpkg

import (
	"bufio"
	"bytes"
	"fmt"
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

func (dpkg DPKG) ListInstalledPackagesSysCall(name string) ([]syspackage.SysPackageInfo, error) {
	// The query format doesn't need shell quoting since exec.Command passes arguments directly.
	format := "${binary:Package},${Version},${Installed-Size}\n"
	argsList := []string{"-W", "-f", format}
	if name != "" {
		argsList = append(argsList, name)
	}
	cmd := exec.Command(dpkg.dpkgquery, argsList...)
	pkgList, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// No packages found, return empty list
			return []syspackage.SysPackageInfo{}, nil
		}
		return nil, fmt.Errorf("dpkg-query command failed: %w, output: %s", err, string(pkgList))
	}

	var lst []syspackage.SysPackageInfo
	scanner := bufio.NewScanner(bytes.NewReader(pkgList))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Trim(line, "'")
		splitLine := strings.Split(line, ",")
		if len(splitLine) != 3 {
			continue
		}
		size, err := strconv.ParseUint(strings.TrimSpace(splitLine[2]), 10, 64)
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

func (dpkg DPKG) QueryPackageSyscall(name string, mode syspackage.QueryMode, lines int) (map[string]any, error) {
	var cmdArgs []string
	var resultKey string
	isInfo := false

	switch mode {
	case syspackage.Info:
		cmdArgs = []string{"-s", name}
		resultKey = "info"
		isInfo = true
	case syspackage.Requires:
		cmdArgs = []string{"-f", "${Depends}", name}
		resultKey = "requires"
	case syspackage.Recommends:
		cmdArgs = []string{"-f", "${Recommends}", name}
		resultKey = "recommends"
	case syspackage.Obsoletes:
		cmdArgs = []string{"-f", "${Breaks}", name}
		resultKey = "obsoletes"
	case syspackage.Changelog:
		cmdArgs = []string{"--changelog", name}
		resultKey = "changelog"
	default:
		return nil, fmt.Errorf("unsupported query mode: %v", mode)
	}

	output, err := exec.Command(dpkg.dpkgquery, cmdArgs...).CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// Package not found
			return nil, fmt.Errorf("package not found: %s", name)
		}
		return nil, fmt.Errorf("failed to query package '%s': %w. Output: %s", name, err, string(output))
	}

	result := make(map[string]any)
	if isInfo {
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
		splittedLines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if lines > 0 && len(splittedLines) > lines {
			result[resultKey] = splittedLines[:lines]
		} else {
			result[resultKey] = splittedLines
		}
	}

	return result, nil
}