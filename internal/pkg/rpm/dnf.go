package rpm

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

func listReposDnf() ([]map[string]any, error) {
	cmd := exec.Command("dnf", "repolist", "-v")
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
