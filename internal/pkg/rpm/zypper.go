package rpm

import (
	"fmt"
	"os/exec"

	"github.com/beevik/etree"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

func (rpm RPM) listReposZypper(params syspackage.ListPackageParams) ([]map[string]any, error) {
	args := []string{"--xmlout", "-s", "0", "lr"}
	if params.Name != "" {
		args = append(args, params.Name)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(output); err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, repoElement := range doc.FindElements("//repo-list/repo") {
		repoMap := make(map[string]any)
		for _, attr := range repoElement.Attr {
			repoMap[attr.Key] = attr.Value
		}
		if urlElement := repoElement.SelectElement("url"); urlElement != nil {
			repoMap["url"] = urlElement.Text()
		}
		result = append(result, repoMap)
	}
	return result, nil
}

func (rpm RPM) modReposZypper(params syspackage.ModifyRepoParams) (map[string]any, error) {
	if params.RemoveRepos {
		cmd := exec.Command(rpm.mgr.mgrpath, "rr", params.Name)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	repos, err := rpm.listReposZypper(syspackage.ListPackageParams{Name: params.Name})
	repoExists := true
	if err != nil {
		repoExists = false
	}
	if repoExists {
		zypperArgs := []string{"mr"}
		if !params.Disable {
			zypperArgs = append(zypperArgs, "-e")
		} else {
			zypperArgs = append(zypperArgs, "-d")
		}
		if params.Name != "" {
			zypperArgs = append(zypperArgs, "-n", params.Name)
		}
		zypperArgs = append(zypperArgs, params.Name)
		cmd := exec.Command(rpm.mgr.mgrpath, zypperArgs...)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	} else {
		cmd := exec.Command(rpm.mgr.mgrpath, "ar", "-f", params.Url, params.Name)
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}

	repos, err = rpm.listReposZypper(syspackage.ListPackageParams{Name: params.Name})
	if err != nil {
		return nil, err
	}
	if len(repos) < 1 {
		return nil, fmt.Errorf("couldn't get repo %s", params.Name)
	} else {
		return repos[0], nil
	}

}

func (rpm RPM) refreshReposZypper(name string) error {
	args := []string{"refresh"}
	if name != "" {
		args = append(args, name)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zypper refresh failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (rpm RPM) listPatchesZypper(params syspackage.ListPatchesParams) ([]map[string]any, error) {
	args := []string{"--xmlout", "lp"}
	if params.Category != "" {
		args = append(args, "--category", params.Category)
	}
	if params.Severity != "" {
		args = append(args, "--severity", params.Severity)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(output); err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, patchElement := range doc.FindElements("//patch-list/patch") {
		patchMap := make(map[string]any)
		for _, attr := range patchElement.Attr {
			patchMap[attr.Key] = attr.Value
		}
		result = append(result, patchMap)
	}
	return result, nil
}

func (rpm RPM) searchPackagesZypper(params syspackage.SearchPackageParams) ([]map[string]any, error) {
	args := []string{"--xmlout", "se", "-s"}
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

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(output); err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, solElement := range doc.FindElements("//solvable-list/solvable") {
		pkgMap := make(map[string]any)
		for _, attr := range solElement.Attr {
			pkgMap[attr.Key] = attr.Value
		}
		result = append(result, pkgMap)
	}
	return result, nil
}

func (rpm RPM) installPatchesZypper(params syspackage.InstallPatchesParams) ([]map[string]any, error) {
	args := []string{"--xmlout", "patch"}
	if params.Category != "" {
		args = append(args, "--category", params.Category)
	}
	if params.Severity != "" {
		args = append(args, "--severity", params.Severity)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(output); err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, patchElement := range doc.FindElements("//patch-list/patch") {
		patchMap := make(map[string]any)
		for _, attr := range patchElement.Attr {
			patchMap[attr.Key] = attr.Value
		}
		result = append(result, patchMap)
	}
	return result, nil
}

func (rpm RPM) installPackageZypper(params syspackage.InstallPackageParams) (string, error) {
	args := []string{"--non-interactive", "install"}
	if params.ShowDetails {
		args = append(args, "--dry-run")
	}
	if params.FromRepo != "" {
		args = append(args, "--from", params.FromRepo)
	}
	if params.WithRecommended {
		args = append(args, "--with-recommended")
	}
	pkg := params.Name
	if params.Version != "" {
		pkg = fmt.Sprintf("%s=%s", params.Name, params.Version)
	}
	args = append(args, pkg)
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("zypper install failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

func (rpm RPM) removePackageZypper(params syspackage.RemovePackageParams) (string, error) {
	args := []string{"--non-interactive", "remove"}
	if params.ShowDetails {
		args = append(args, "--dry-run")
	}
	if params.RemoveDeps {
		args = append(args, "--clean-deps")
	}
	args = append(args, params.Name)
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("zypper remove failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

func (rpm RPM) updatePackageZypper(params syspackage.UpdatePackageParams) (string, error) {
	args := []string{"--non-interactive", "update"}
	if len(params.Repos) > 0 {
		for _, repo := range params.Repos {
			args = append(args, "--from", repo)
		}
	}
	if params.Name != "" {
		args = append(args, params.Name)
	}
	cmd := exec.Command(rpm.mgr.mgrpath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("zypper update failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}
