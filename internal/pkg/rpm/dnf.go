package rpm

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

func (rpm RPM) listReposDnf(params syspackage.ListPackageParams) ([]map[string]any, error) {
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "repo", "list")
	if params.Name != "" {
		args = append(args, params.Name)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	var repos []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(output))
	currentRepo := make(map[string]any)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if len(currentRepo) > 0 {
				repos = append(repos, currentRepo)
				currentRepo = make(map[string]any)
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			currentRepo[key] = value
		}
	}

	if len(currentRepo) > 0 {
		repos = append(repos, currentRepo)
	}

	return repos, nil

}

func (rpm RPM) modReposDnf(params syspackage.ModifyRepoParams) (map[string]any, error) {
	if params.RemoveRepos {
		args := []string{}
		if rpm.root != "" {
			args = append(args, "--root", rpm.root)
		}
		args = append(args, "repo", "remove", params.Name)
		cmd := exec.Command(rpm.mgr.mgrpath, args...)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "repo", "modify")
	if !params.Disable {
		args = append(args, "--enable")
	} else {
		args = append(args, "--disable")
	}
	if params.Name != "" {
		args = append(args, "--name", params.Name)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	err := cmd.Run()
	if err != nil {
		// if the repo does not exist, add it
		args := []string{}
		if rpm.root != "" {
			args = append(args, "--root", rpm.root)
		}
		args = append(args, "config-manager", "--add-repo", params.Url)
		cmd := exec.Command(rpm.mgr.mgrpath, args...)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}

	repos, err := rpm.listReposDnf(syspackage.ListPackageParams{Name: params.Name})
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		if r, ok := repo["Repo-id"]; ok && r == params.Name {
			return repo, nil
		}
	}

	return nil, nil
}

func (rpm RPM) refreshReposDnf(name string) error {
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "makecache")
	if name != "" {
		args = append(args, "--disablerepo='*'", fmt.Sprintf("--enablerepo='%s'", name))
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf makecache failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (rpm RPM) searchPackagesDnf(params syspackage.SearchPackageParams) (map[string]map[string][]syspackage.SearchedPackage, error) {
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "repoquery", "--queryformat", "%{name}\t%{repoid}\t%{arch}\t%{version}-%{release}")
	if len(params.Repos) > 0 {
		for _, repo := range params.Repos {
			args = append(args, "--repo", repo)
		}
	}

	query := params.Name
	if !strings.Contains(query, "*") && !strings.Contains(query, "?") {
		query = "*" + query + "*"
	}
	args = append(args, query)
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	result := make(map[string]map[string][]syspackage.SearchedPackage)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return result, nil
		}
		return nil, fmt.Errorf("dnf repoquery failed: %w, output: %s", err, string(output))
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			continue
		}
		name := parts[0]
		repo := parts[1]
		arch := parts[2]
		version := parts[3]

		status := "v"
		if repo == "@System" {
			repo = "System"
			status = "i"
		}
		if repo == "" {
			repo = "unknown"
		}
		if arch == "" {
			arch = "unknown"
		}

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

	return result, nil
}

func (rpm RPM) installPackageDnf(params syspackage.InstallPackageParams) (string, error) {
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "install")
	if params.ShowDetails {
		args = append(args, "--assumeno")
	} else {
		args = append(args, "-y")
	}
	if params.FromRepo != "" {
		args = append(args, "--repo", params.FromRepo)
	}
	if params.WithRecommended {
		args = append(args, "--setopt=install_weak_deps=True")
	} else {
		args = append(args, "--setopt=install_weak_deps=False")
	}
	pkg := params.Name
	if params.Version != "" {
		pkg = fmt.Sprintf("%s-%s", params.Name, params.Version)
	}
	args = append(args, pkg)
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("dnf install failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

func (rpm RPM) removePackageDnf(params syspackage.RemovePackageParams) (string, error) {
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "remove")
	if params.ShowDetails {
		args = append(args, "--assumeno")
	} else {
		args = append(args, "-y")
	}
	if params.Purge {
		args = append(args, "--purge")
	}
	if params.RemoveDeps {
		args = append(args, "--setopt=clean_requirements_on_remove=True")
	}
	args = append(args, params.Name)
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("dnf remove failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

func (rpm RPM) updatePackageDnf(params syspackage.UpdatePackageParams) (string, error) {
	args := []string{}
	if rpm.root != "" {
		args = append(args, "--root", rpm.root)
	}
	args = append(args, "upgrade", "-y")
	if len(params.Repos) > 0 {
		for _, repo := range params.Repos {
			args = append(args, "--repo", repo)
		}
	}
	if params.Name != "" {
		args = append(args, params.Name)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("dnf upgrade failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}
