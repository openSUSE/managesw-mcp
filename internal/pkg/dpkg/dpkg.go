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
	aptcache  string
}

func New(dpkgbin string, dpkgquery string, aptcache string) DPKG {
	return DPKG{
		dpkgbin:   dpkgbin,
		dpkgquery: dpkgquery,
		aptcache:  aptcache,
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

func (dpkg DPKG) QueryPackageSysCall(name string, mode syspackage.QueryMode, lines int) (map[string]any, error) {
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
func (dpkg DPKG) ListReposSysCall(name string) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpkg DPKG) ModifyRepoSysCall(params syspackage.ModifyRepoParams) (map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpkg DPKG) ListPatchesSysCall(params syspackage.ListPatchesParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpkg DPKG) InstallPatchesSysCall(params syspackage.InstallPatchesParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpkg DPKG) RefreshReposSysCall(name string) error {
	return fmt.Errorf("not implemented")
}

func (dpkg DPKG) SearchPackageSysCall(params syspackage.SearchPackageParams) (any, error) {
	aptcache := dpkg.aptcache
	if aptcache == "" {
		var err error
		aptcache, err = exec.LookPath("apt-cache")
		if err != nil {
			return nil, fmt.Errorf("apt-cache binary not found: %w", err)
		}
	}

	// First search for package names using apt-cache search
	cmd := exec.Command(aptcache, "search", "--names-only", params.Name)
	output, err := cmd.CombinedOutput()
	result := make(map[string]map[string][]syspackage.SearchedPackage)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return result, nil
		}
		return nil, fmt.Errorf("apt-cache search failed: %w, output: %s", err, string(output))
	}

	var pkgNames []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) > 0 {
			pkgName := strings.TrimSpace(parts[0])
			if pkgName != "" {
				pkgNames = append(pkgNames, pkgName)
			}
		}
	}

	if len(pkgNames) == 0 {
		return result, nil
	}

	// Cap search to a reasonable number to prevent calling madison with thousands of arguments
	if len(pkgNames) > 200 {
		pkgNames = pkgNames[:200]
	}

	// Run apt-cache madison to get structured version and repository info
	args := append([]string{"madison"}, pkgNames...)
	cmdMadison := exec.Command(aptcache, args...)
	madisonOutput, err := cmdMadison.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			madisonOutput = []byte{}
		} else {
			return nil, fmt.Errorf("apt-cache madison failed: %w, output: %s", err, string(madisonOutput))
		}
	}

	// Query installed packages with matching pattern to determine status
	installedMap := make(map[string]string)
	queryName := params.Name
	if !strings.Contains(queryName, "*") && !strings.Contains(queryName, "?") {
		queryName = "*" + queryName + "*"
	}
	installedPkgs, err := dpkg.ListInstalledPackagesSysCall(queryName)
	if err == nil {
		for _, p := range installedPkgs {
			installedMap[p.Name] = p.Version
		}
	}

	// Parse madison output
	scannerMadison := bufio.NewScanner(bytes.NewReader(madisonOutput))
	for scannerMadison.Scan() {
		line := scannerMadison.Text()
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		sourceStr := strings.TrimSpace(parts[2])

		sourceFields := strings.Fields(sourceStr)
		repo := "unknown"
		arch := "unknown"
		if len(sourceFields) >= 3 {
			repo = sourceFields[0]
			arch = sourceFields[2]
		} else if len(sourceFields) > 0 {
			repo = sourceFields[0]
		}

		// Filter by requested repositories if supplied
		if len(params.Repos) > 0 {
			matched := false
			for _, r := range params.Repos {
				if strings.Contains(strings.ToLower(repo), strings.ToLower(r)) || strings.Contains(strings.ToLower(r), strings.ToLower(repo)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		status := "v"

		if _, exists := result[repo]; !exists {
			result[repo] = make(map[string][]syspackage.SearchedPackage)
		}

		pkg := syspackage.SearchedPackage{
			Name:    name,
			Version: version,
			Status:  status,
		}
		result[repo][arch] = append(result[repo][arch], pkg)
	}

	// Add installed packages to "System" repository to mirror zypper behavior
	for instName, instVer := range installedMap {
		repo := "System"
		arch := "unknown"

		// Filter system packages if Repos was specified (and didn't include system)
		if len(params.Repos) > 0 {
			matched := false
			for _, r := range params.Repos {
				if strings.Contains(strings.ToLower(repo), strings.ToLower(r)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if _, exists := result[repo]; !exists {
			result[repo] = make(map[string][]syspackage.SearchedPackage)
		}

		pkg := syspackage.SearchedPackage{
			Name:    instName,
			Version: instVer,
			Status:  "i",
		}
		result[repo][arch] = append(result[repo][arch], pkg)
	}

	return result, nil
}

func (dpkg DPKG) InstallPackageSysCall(params syspackage.InstallPackageParams) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (dpkg DPKG) RemovePackageSysCall(params syspackage.RemovePackageParams) (string, error) {
	if params.Name == "" {
		return "", fmt.Errorf("package name is required")
	}

	var cmdArgs []string
	if params.Purge {
		cmdArgs = append(cmdArgs, "--purge")
	} else {
		cmdArgs = append(cmdArgs, "--remove")
	}

	if params.ShowDetails {
		cmdArgs = append(cmdArgs, "--dry-run")
	}

	cmdArgs = append(cmdArgs, params.Name)

	cmd := exec.Command(dpkg.dpkgbin, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to remove package '%s': %w. Output: %s", params.Name, err, string(output))
	}

	return string(output), nil
}

func (dpkg DPKG) PkgType() string {
	return "dpkg"
}

func (dpkg DPKG) UpdatePackageSysCall(params syspackage.UpdatePackageParams) (string, error) {
	return "", fmt.Errorf("not implemented")
}
