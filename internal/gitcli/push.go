package gitcli

import (
	"errors"
	"fmt"
	"strings"
)

type PushResult struct {
	Branch      string `json:"branch"`
	Remote      string `json:"remote"`
	Upstream    string `json:"upstream"`
	SetUpstream bool   `json:"setUpstream"`
	Rejected    bool   `json:"rejected"`
	Detail      string `json:"detail"`
}

// PushBranch pushes branch (empty = the current branch) to its upstream, or to the default
// remote with -u when it has none. A non-fast-forward rejection is reported in the result
// rather than as an error, so the caller can offer a force push.
//
// A bare --force-with-lease compares against the remote-tracking ref, which background
// auto-fetch keeps current, so on its own it would overwrite a teammate's commits that were
// merely fetched. --force-if-includes additionally requires the remote tip to be part of the
// local branch's history (its reflog), so only commits you actually had can be replaced.
func PushBranch(repoPath, branch string, force bool) (PushResult, error) {
	var res PushResult
	if branch == "" {
		out, err := Run(repoPath, "symbolic-ref", "--quiet", "--short", "HEAD")
		if err != nil {
			return res, errors.New("HEAD is detached; check out a branch to push it")
		}
		branch = strings.TrimSpace(out.Stdout)
	}
	res.Branch = branch

	remote := configValue(repoPath, "branch."+branch+".remote")
	mergeRef := configValue(repoPath, "branch."+branch+".merge")
	args := []string{"push", "--porcelain"}
	if force {
		args = append(args, "--force-with-lease", "--force-if-includes")
	}
	if remote != "" && mergeRef != "" {
		res.Remote = remote
		res.Upstream = remote + "/" + strings.TrimPrefix(mergeRef, "refs/heads/")
		args = append(args, remote, "refs/heads/"+branch+":"+mergeRef)
	} else {
		def, err := defaultRemote(repoPath)
		if err != nil {
			return res, err
		}
		res.Remote = def
		res.Upstream = def + "/" + branch
		res.SetUpstream = true
		args = append(args, "--set-upstream", def, "refs/heads/"+branch+":refs/heads/"+branch)
	}

	out, err := Run(repoPath, args...)
	if err == nil {
		return res, nil
	}
	text := out.Stdout + "\n" + out.Stderr
	if !force && isNonFastForward(text) {
		res.Rejected = true
		res.Detail = strings.TrimSpace(out.Stderr)
		return res, nil
	}
	if force && (strings.Contains(text, "stale info") || strings.Contains(text, "remote ref updated since checkout")) {
		return res, fmt.Errorf("%s has commits that aren't in your branch's history, so the force push was refused. Pull or review them first, then try again", res.Upstream)
	}
	return res, err
}

func isNonFastForward(text string) bool {
	return strings.Contains(text, "[rejected]") &&
		(strings.Contains(text, "non-fast-forward") || strings.Contains(text, "fetch first"))
}

func configValue(repoPath, key string) string {
	out, err := Run(repoPath, "config", "--get", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out.Stdout)
}

func defaultRemote(repoPath string) (string, error) {
	out, err := Run(repoPath, "remote")
	if err != nil {
		return "", err
	}
	remotes := strings.Fields(out.Stdout)
	if len(remotes) == 0 {
		return "", errors.New("this repository has no remote to push to; add one with git remote add")
	}
	for _, r := range remotes {
		if r == "origin" {
			return r, nil
		}
	}
	return remotes[0], nil
}
