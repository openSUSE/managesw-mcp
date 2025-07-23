package rpm

import (
	"os/exec"

	"github.com/beevik/etree"
)

func listReposZypper() ([]map[string]any, error) {
	cmd := exec.Command("zypper", "--xmlout", "-s", "0", "lr", "-d")
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
