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