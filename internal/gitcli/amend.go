package gitcli

import "strings"

type AmendInfo struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Pushed      bool   `json:"pushed"`
}

// Pushed means HEAD is already on its upstream, so amending rewrites published history.
func GetAmendInfo(repoPath string) (AmendInfo, error) {
	var info AmendInfo
	out, err := Run(repoPath, "log", "-1", "--pretty=%B")
	if err != nil {
		return info, err
	}
	summary, rest, _ := strings.Cut(strings.TrimRight(out.Stdout, "\n"), "\n")
	info.Summary = strings.TrimSpace(summary)
	info.Description = strings.TrimSpace(rest)
	if _, err := Run(repoPath, "merge-base", "--is-ancestor", "HEAD", "@{upstream}"); err == nil {
		info.Pushed = true
	}
	return info, nil
}
