package rpm

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path"
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
	root    string
	isTest  bool
}

func NewRPM(path string, systype RPMType, mgrpath string, root string) RPM {
	return RPM{
		rpmpath: path,
		mgr: PkgMgr{
			mgrtype: systype,
			mgrpath: mgrpath,
		},
		root: root,
	}
}

func NewRPMTest(path string, systype RPMType, mgrpath string, root string) RPM {
	return RPM{
		rpmpath: path,
		mgr: PkgMgr{
			mgrtype: systype,
			mgrpath: mgrpath,
		},
		root:   root,
		isTest: true,
	}
}

// ListInstalledPackagesSysCall lists the installed packages given by their name pattern.
func (rpm RPM) ListInstalledPackagesSysCall(params syspackage.ListPackageParams) ([]syspackage.SysPackageInfo, error) {
	// The query format doesn't need shell quoting since exec.Command passes arguments directly.
	qf := `%{NAME},%{VERSION},%{SIZE}\n`
	args := []string{}
	if rpm.isTest {
		args = append(args, "--dbpath", path.Join(rpm.root, "/var/lib/rpm"))
	} else if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "-qa", "--qf", qf)
	if params.Name != "" {
		args = append(args, params.Name)
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

	// Fetch additional fields if requested
	for i := range lst {
		pkgName := lst[i].Name
		if params.Filelist {
			fileArgs := []string{}
			if rpm.isTest {
				fileArgs = append(fileArgs, "--dbpath", path.Join(rpm.root, "/var/lib/rpm"))
			} else if rpm.root != "" {
				fileArgs = append(fileArgs, "--root", rpm.root)
			}
			fileArgs = append(fileArgs, "-ql", pkgName)
			fileListOut, err := exec.Command(rpm.rpmpath, fileArgs...).CombinedOutput()
			if err == nil {
				scannerFiles := bufio.NewScanner(bytes.NewReader(fileListOut))
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
			descArgs := []string{}
			if rpm.isTest {
				descArgs = append(descArgs, "--dbpath", path.Join(rpm.root, "/var/lib/rpm"))
			} else if rpm.root != "" {
				descArgs = append(descArgs, "--root", rpm.root)
			}
			descArgs = append(descArgs, "-q", "--qf", "%{DESCRIPTION}", pkgName)
			descOut, err := exec.Command(rpm.rpmpath, descArgs...).CombinedOutput()
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
				var relFlag string
				switch rel {
				case "requires":
					relFlag = "--requires"
				case "recommends":
					relFlag = "--recommends"
				case "obsoletes":
					relFlag = "--obsoletes"
				case "provides":
					relFlag = "--provides"
				case "conflicts":
					relFlag = "--conflicts"
				case "suggests":
					relFlag = "--suggests"
				case "supplements":
					relFlag = "--supplements"
				case "enhances":
					relFlag = "--enhances"
				default:
					continue
				}

				relArgs := []string{}
				if rpm.isTest {
					relArgs = append(relArgs, "--dbpath", path.Join(rpm.root, "/var/lib/rpm"))
				} else if rpm.root != "" {
					relArgs = append(relArgs, "--root", rpm.root)
				}
				relArgs = append(relArgs, "-q", relFlag, pkgName)
				relOut, err := exec.Command(rpm.rpmpath, relArgs...).CombinedOutput()
				if err == nil {
					scannerRel := bufio.NewScanner(bytes.NewReader(relOut))
					var rels []string
					for scannerRel.Scan() {
						r := strings.TrimSpace(scannerRel.Text())
						if r != "" && !strings.HasPrefix(r, "package ") && !strings.Contains(r, "is not installed") {
							rels = append(rels, r)
						}
					}
					lst[i].Relations[rel] = rels
				}
			}
		}

		if params.Changelog > 0 {
			changeArgs := []string{}
			if rpm.isTest {
				changeArgs = append(changeArgs, "--dbpath", path.Join(rpm.root, "/var/lib/rpm"))
			} else if rpm.root != "" {
				changeArgs = append(changeArgs, "--root", rpm.root)
			}
			changeArgs = append(changeArgs, "-q", "--changelog", pkgName)
			changeOut, err := exec.Command(rpm.rpmpath, changeArgs...).CombinedOutput()
			if err == nil {
				scannerChange := bufio.NewScanner(bytes.NewReader(changeOut))
				var lines []string
				var count uint
				for scannerChange.Scan() && count < params.Changelog {
					lines = append(lines, scannerChange.Text())
					count++
				}
				lst[i].Changelog = strings.Join(lines, "\n")
			}
		}
	}

	return lst, nil
}

// QueryPackageSyscall queries package information.
func (rpm RPM) QueryPackageSysCall(name string, mode syspackage.QueryMode, lines int) (result map[string]any, err error) {
	var cmdArgs []string
	var resultKey string

	if rpm.isTest {
		cmdArgs = append(cmdArgs, "--dbpath", path.Join(rpm.root, "/var/lib/rpm"))
	} else if rpm.root != "" {
		cmdArgs = append(cmdArgs, "--root", rpm.root)
	}

	switch mode {
	case syspackage.Info:
		cmdArgs = append(cmdArgs, "-qi", name)
		resultKey = "info"
	case syspackage.Requires:
		cmdArgs = append(cmdArgs, "-q", "--requires", name)
		resultKey = "requires"
	case syspackage.Recommends:
		cmdArgs = append(cmdArgs, "-q", "--recommends", name)
		resultKey = "recommends"
	case syspackage.Obsoletes:
		cmdArgs = append(cmdArgs, "-q", "--obsoletes", name)
		resultKey = "obsoletes"
	case syspackage.Changelog:
		cmdArgs = append(cmdArgs, "-q", "--changelog", name)
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

func (rpm RPM) ListReposSysCall(name string) ([]map[string]any, error) {
	params := syspackage.ListPackageParams{Name: name}
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.listReposZypper(params)
	case Dnf:
		return rpm.listReposDnf(params)
	default:
		return nil, fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) ModifyRepoSysCall(params syspackage.ModifyRepoParams) (ret map[string]any, err error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.modReposZypper(params)
	case Dnf:
		return rpm.modReposDnf(params)
	default:
		return nil, fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) RefreshReposSysCall(name string) error {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.refreshReposZypper(name)
	case Dnf:
		return rpm.refreshReposDnf(name)
	default:
		return fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) ListPatchesSysCall(params syspackage.ListPatchesParams) ([]map[string]any, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.listPatchesZypper(params)
	case Dnf:
		return nil, fmt.Errorf("Listing patches is not supported on dnf")
	default:
		return nil, fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) InstallPatchesSysCall(params syspackage.InstallPatchesParams) ([]map[string]any, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.installPatchesZypper(params)
	case Dnf:
		return nil, fmt.Errorf("Installing patches is not supported on dnf")
	default:
		return nil, fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) SearchPackageSysCall(params syspackage.SearchPackageParams) (any, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.searchPackagesZypper(params)
	case Dnf:
		return rpm.searchPackagesDnf(params)
	default:
		return nil, fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) InstallPackageSysCall(params syspackage.InstallPackageParams) (string, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.installPackageZypper(params)
	case Dnf:
		return rpm.installPackageDnf(params)
	default:
		return "", fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) RemovePackageSysCall(params syspackage.RemovePackageParams) (string, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.removePackageZypper(params)
	case Dnf:
		return rpm.removePackageDnf(params)
	default:
		return "", fmt.Errorf("No rpm package manager installed")
	}
}

func (rpm RPM) PkgType() string {
	return "rpm"
}

func (rpm RPM) UpdatePackageSysCall(params syspackage.UpdatePackageParams) (string, error) {
	switch rpm.mgr.mgrtype {
	case Zypper:
		return rpm.updatePackageZypper(params)
	case Dnf:
		return rpm.updatePackageDnf(params)
	default:
		return "", fmt.Errorf("No rpm package manager installed")
	}
}
