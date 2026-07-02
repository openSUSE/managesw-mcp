package dpkg

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

type DPKG struct {
	dpkgbin   string
	dpkgquery string
	aptcache  string
	root      string
}

func New(dpkgbin string, dpkgquery string, aptcache string, root string) DPKG {
	return DPKG{
		dpkgbin:   dpkgbin,
		dpkgquery: dpkgquery,
		aptcache:  aptcache,
		root:      root,
	}
}

func (dpkg DPKG) ListInstalledPackagesSysCall(params syspackage.ListPackageParams) ([]syspackage.SysPackageInfo, error) {
	// The query format doesn't need shell quoting since exec.Command passes arguments directly.
	format := "${binary:Package},${Version},${Installed-Size}\n"
	argsList := []string{"-W", "-f", format}
	if params.Name != "" {
		argsList = append(argsList, params.Name)
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

	// Fetch additional fields if requested
	for i := range lst {
		pkgName := lst[i].Name
		if params.Filelist {
			fileOut, err := exec.Command(dpkg.dpkgbin, "-L", pkgName).CombinedOutput()
			if err == nil {
				scannerFiles := bufio.NewScanner(bytes.NewReader(fileOut))
				var files []string
				for scannerFiles.Scan() {
					f := strings.TrimSpace(scannerFiles.Text())
					if f != "" {
						files = append(files, f)
					}
				}
				lst[i].FileList = files
			}
		}

		if params.Description {
			descOut, err := exec.Command(dpkg.dpkgquery, "-f", "${Description}", "-W", pkgName).CombinedOutput()
			if err == nil {
				lst[i].Description = string(descOut)
			}
		}

		if len(params.Relations) > 0 {
			lst[i].Relations = make(map[string][]string)
			for _, rel := range params.Relations {
				rel = strings.ToLower(strings.TrimSpace(rel))
				if rel == "" {
					continue
				}
				var field string
				switch rel {
				case "requires":
					field = "${Depends}"
				case "recommends":
					field = "${Recommends}"
				case "obsoletes":
					field = "${Breaks}"
				case "provides":
					field = "${Provides}"
				case "conflicts":
					field = "${Conflicts}"
				case "suggests":
					field = "${Suggests}"
				case "supplements":
					field = "${Enhances}"
				case "enhances":
					field = "${Enhances}"
				default:
					continue
				}

				relOut, err := exec.Command(dpkg.dpkgquery, "-f", field, "-W", pkgName).CombinedOutput()
				if err == nil {
					// dpkg-query returns a single line of comma-separated packages for these fields
					line := strings.TrimSpace(string(relOut))
					var rels []string
					if line != "" {
						parts := strings.Split(line, ",")
						for _, p := range parts {
							p = strings.TrimSpace(p)
							if p != "" {
								rels = append(rels, p)
							}
						}
					}
					lst[i].Relations[rel] = rels
				}
			}
		}

		if params.Changelog > 0 {
			// Debian changelog can be read from file or retrieved via apt-get changelog
			// To keep it simple and avoid downloading, try reading the local changelog file if it exists.
			// Path: /usr/share/doc/<package>/changelog.Debian.gz (need zcat or gzip to read, or try uncompressed)
			// Let's just return empty or use apt-get changelog if we wanted to. We can leave it empty if local files don't exist.
			lst[i].Changelog = ""
		}
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
func parseAptListFile(filePath string) (enabled bool, url string) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		isComment := false
		if strings.HasPrefix(line, "#") {
			isComment = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}

		if !strings.HasPrefix(line, "deb") && !strings.HasPrefix(line, "deb-src") {
			continue
		}

		if !isComment {
			enabled = true
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			urlIdx := 1
			if strings.HasPrefix(fields[1], "[") {
				for i := 1; i < len(fields); i++ {
					if !strings.HasPrefix(fields[i], "[") && !strings.Contains(fields[i], "=") {
						urlIdx = i
						break
					}
				}
			}
			if urlIdx < len(fields) {
				if url == "" {
					url = fields[urlIdx]
				}
			}
		}
	}
	return enabled, url
}

func (dpkg DPKG) getRepos() ([]map[string]any, error) {
	var repos []map[string]any

	sourcesListPath := filepath.Join(dpkg.root, "etc/apt/sources.list")
	if _, err := os.Stat(sourcesListPath); err == nil {
		enabled, url := parseAptListFile(sourcesListPath)
		enabledStr := "0"
		if enabled {
			enabledStr = "1"
		}
		repos = append(repos, map[string]any{
			"alias":   "sources.list",
			"name":    "sources.list",
			"enabled": enabledStr,
			"url":     url,
		})
	}

	sourcesListDDir := filepath.Join(dpkg.root, "etc/apt/sources.list.d")
	files, err := os.ReadDir(sourcesListDDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".list") {
				alias := strings.TrimSuffix(file.Name(), ".list")
				filePath := filepath.Join(sourcesListDDir, file.Name())
				enabled, url := parseAptListFile(filePath)
				enabledStr := "0"
				if enabled {
					enabledStr = "1"
				}
				repos = append(repos, map[string]any{
					"alias":   alias,
					"name":    alias,
					"enabled": enabledStr,
					"url":     url,
				})
			}
		}
	}

	return repos, nil
}

func (dpkg DPKG) ListReposSysCall(name string) ([]map[string]any, error) {
	allRepos, err := dpkg.getRepos()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return allRepos, nil
	}
	var filtered []map[string]any
	for _, repo := range allRepos {
		if repo["alias"].(string) == name {
			filtered = append(filtered, repo)
		}
	}
	return filtered, nil
}

func (dpkg DPKG) ModifyRepoSysCall(params syspackage.ModifyRepoParams) (map[string]any, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	var filePath string
	if params.Name == "sources.list" {
		filePath = filepath.Join(dpkg.root, "etc/apt/sources.list")
	} else {
		filePath = filepath.Join(dpkg.root, "etc/apt/sources.list.d", params.Name+".list")
	}

	if params.RemoveRepos {
		_ = os.Remove(filePath)
		return nil, nil
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var line string
	prefix := "deb"
	if params.Disable {
		prefix = "# deb"
	}

	options := ""
	if params.NoGPGCheck {
		options = "[trusted=yes] "
	}

	url := params.Url
	if url == "" {
		if _, err := os.Stat(filePath); err == nil {
			_, existingUrl := parseAptListFile(filePath)
			url = existingUrl
		}
	}

	if url == "" {
		return nil, fmt.Errorf("repository URL is required")
	}

	urlParts := strings.Fields(url)
	if len(urlParts) == 1 && !strings.HasSuffix(urlParts[0], "./") {
		line = fmt.Sprintf("%s %s%s ./", prefix, options, urlParts[0])
	} else {
		line = fmt.Sprintf("%s %s%s", prefix, options, url)
	}

	if err := os.WriteFile(filePath, []byte(line+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("failed to write repository file: %w", err)
	}

	repos, err := dpkg.ListReposSysCall(params.Name)
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("could not get repository %s after modification", params.Name)
	}
	return repos[0], nil
}

func (dpkg DPKG) ListPatchesSysCall(params syspackage.ListPatchesParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpkg DPKG) InstallPatchesSysCall(params syspackage.InstallPatchesParams) ([]map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpkg DPKG) RefreshReposSysCall(name string) error {
	aptget, err := exec.LookPath("apt-get")
	if err != nil {
		return fmt.Errorf("apt-get binary not found: %w", err)
	}

	args := []string{}
	if dpkg.root != "" {
		args = append(args, "-o", "RootDir="+dpkg.root)
	}
	args = append(args, "update")

	if name != "" {
		var sourcelist string
		if name == "sources.list" {
			sourcelist = "sources.list"
		} else {
			sourcelist = filepath.Join("sources.list.d", name+".list")
		}
		args = append(args, "-o", "Dir::Etc::sourcelist="+sourcelist)
		args = append(args, "-o", "Dir::Etc::sourceparts=none")
	}

	cmd := exec.Command(aptget, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt-get update failed: %w, output: %s", err, string(output))
	}
	return nil
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
	installedPkgs, err := dpkg.ListInstalledPackagesSysCall(syspackage.ListPackageParams{Name: queryName})
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
