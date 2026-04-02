package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Commit struct {
	Hash    string
	Author  string
	Date    string
	Message string
	Applied bool
}

func compareByMessage(source, target, mergeBase string) ([]Commit, error) {
	sourceLog, err := git("log", "--pretty=format:%h\t%an\t%ad\t%s", "--date=format:%Y-%m-%d %H:%M", mergeBase+".."+source)
	if err != nil {
		return nil, fmt.Errorf("reading source commits: %w", err)
	}

	targetLog, err := git("log", "--pretty=format:%s", mergeBase+".."+target)
	if err != nil {
		return nil, fmt.Errorf("reading target commits: %w", err)
	}

	targetMessages := make(map[string]struct{})
	for _, msg := range strings.Split(strings.TrimSpace(targetLog), "\n") {
		if msg != "" {
			targetMessages[strings.TrimSpace(msg)] = struct{}{}
		}
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(sourceLog), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		msg := strings.TrimSpace(parts[3])
		_, applied := targetMessages[msg]
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  strings.TrimSpace(parts[1]),
			Date:    strings.TrimSpace(parts[2]),
			Message: msg,
			Applied: applied,
		})
	}
	return commits, nil
}

func filterCommits(commits []Commit, applied bool) []Commit {
	var out []Commit
	for _, c := range commits {
		if c.Applied == applied {
			out = append(out, c)
		}
	}
	return out
}

func getMergeBase(a, b string) (string, error) {
	out, err := git("merge-base", a, b)
	if err != nil {
		return "", err
	}
	hash := strings.TrimSpace(out)
	if len(hash) > 7 {
		hash = hash[:7]
	}
	return hash, nil
}

func validateBranch(branch string) error {
	_, err := git("rev-parse", "--verify", branch)
	return err
}

func checkGitRepo() error {
	_, err := git("rev-parse", "--git-dir")
	return err
}

func getCurrentBranch() (string, error) {
	out, err := git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func cherryPickInProgress() bool {
	out, err := git("rev-parse", "--git-dir")
	if err != nil {
		return false
	}
	gitDir := strings.TrimSpace(out)
	_, err1 := os.Stat(gitDir + "/CHERRY_PICK_HEAD")
	_, err2 := os.Stat(gitDir + "/sequencer")
	return err1 == nil || err2 == nil
}

func getCommitFiles(hash string) ([]string, error) {
	out, err := git("diff-tree", "--no-commit-id", "-r", "--name-only", hash)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
