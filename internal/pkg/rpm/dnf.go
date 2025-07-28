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
	args := []string{"repo", "list"}
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
		cmd := exec.Command(rpm.mgr.mgrpath, "repo", "remove", params.Name)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	args := []string{"repo", "modify"}
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
		cmd := exec.Command(rpm.mgr.mgrpath, "config-manager", "--add-repo", params.Url)
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
	args := []string{"makecache"}
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

func (rpm RPM) searchPackagesDnf(params syspackage.SearchPackageParams) ([]map[string]any, error) {
	args := []string{"search"}
	if len(params.Repos) > 0 {
		for _, repo := range params.Repos {
			args = append(args, "--repo", repo)
		}
	}
	args = append(args, params.Name)
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var packages []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentPackage map[string]any
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "=======") {
			if currentPackage != nil {
				packages = append(packages, currentPackage)
			}
			currentPackage = make(map[string]any)
		} else if currentPackage != nil {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				currentPackage[key] = value
			}
		}
	}
	if currentPackage != nil {
		packages = append(packages, currentPackage)
	}

	return packages, nil
}
