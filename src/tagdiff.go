package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runTagDiff() {
	var (
		flagShowAuthor = flag.String("show-author", "", "show commit author: true|false")
		flagShowDate   = flag.String("show-date", "", "show commit date: true|false")
		flagFormat     = flag.String("format", "", "output format: default|json")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gitmt tagdiff [from] <to|latest>\n\nArgs:\n  from     Starting tag (omit to use the tag before <to>)\n  to       Ending tag, or \"latest\" to resolve to the most recent tag\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	var from, to string

	switch len(args) {
	case 1:
		to = args[0]
		if to == "latest" {
			latest, err := getLatestTag()
			if err != nil {
				fatalf("could not resolve latest tag: %v", err)
			}
			to = latest
		}
		prev, err := getPreviousTag(to)
		if err != nil {
			fatalf("could not find a tag before %q: %v", to, err)
		}
		from = prev
	case 2:
		from = args[0]
		to = args[1]
	default:
		flag.Usage()
		os.Exit(1)
	}

	if err := checkGitRepo(); err != nil {
		fatalf("not inside a git repository: %v", err)
	}
	if err := validateBranch(from); err != nil {
		fatalf("tag %q not found: %v", from, err)
	}
	if err := validateBranch(to); err != nil {
		fatalf("tag %q not found: %v", to, err)
	}

	cfg := loadConfig()
	if *flagShowAuthor != "" {
		v := *flagShowAuthor == "true"
		cfg.ShowAuthor = &v
	}
	if *flagShowDate != "" {
		v := *flagShowDate == "true"
		cfg.ShowDate = &v
	}
	if *flagFormat != "" {
		cfg.Format = *flagFormat
	}
	showAuthor := cfg.ShowAuthor == nil || *cfg.ShowAuthor
	showDate := cfg.ShowDate == nil || *cfg.ShowDate

	commits, err := getTagCommits(from, to)
	if err != nil {
		fatalf("could not get commits: %v", err)
	}

	if cfg.Format == "json" {
		printTagDiffJSON(from, to, commits, showDate)
	} else {
		printTagDiffOutput(from, to, commits, showAuthor, showDate)
	}
}

func getLatestTag() (string, error) {
	out, err := git("tag", "--sort=-version:refname")
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no tags found in repository")
	}
	return strings.TrimSpace(lines[0]), nil
}

func getPreviousTag(tag string) (string, error) {
	out, err := git("describe", "--tags", "--abbrev=0", tag+"^")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func getTagCommits(from, to string) ([]Commit, error) {
	out, err := git("log", "--pretty=format:%h\t%an\t%ad\t%s", "--date=format:%Y-%m-%d %H:%M", from+".."+to)
	if err != nil {
		return nil, err
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  strings.TrimSpace(parts[1]),
			Date:    strings.TrimSpace(parts[2]),
			Message: strings.TrimSpace(parts[3]),
		})
	}
	return commits, nil
}

type tagDiffJSON struct {
	From    string           `json:"from"`
	To      string           `json:"to"`
	Total   int              `json:"total"`
	Commits []tagDiffCommit  `json:"commits"`
}

type tagDiffCommit struct {
	Hash    string `json:"hash"`
	Date    string `json:"date,omitempty"`
	Author  string `json:"author"`
	Message string `json:"message"`
}

func printTagDiffJSON(from, to string, commits []Commit, showDate bool) {
	jCommits := make([]tagDiffCommit, len(commits))
	for i, c := range commits {
		jc := tagDiffCommit{
			Hash:    c.Hash,
			Author:  c.Author,
			Message: c.Message,
		}
		if showDate {
			jc.Date = c.Date
		}
		jCommits[i] = jc
	}
	out := tagDiffJSON{
		From:    from,
		To:      to,
		Total:   len(commits),
		Commits: jCommits,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func printTagDiffOutput(from, to string, commits []Commit, showAuthor, showDate bool) {
	fmt.Printf("\n%s%sTag diff%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s  from  : %s%s\n", colorDim, colorReset, from)
	fmt.Printf("%s  to    : %s%s\n", colorDim, colorReset, to)
	fmt.Printf("%s  total : %s%d commit(s)\n", colorDim, colorReset, len(commits))

	if len(commits) == 0 {
		fmt.Printf("\n%sNo commits between %s and %s.%s\n\n", colorDim, from, to, colorReset)
		return
	}

	fmt.Println()
	for i, c := range commits {
		fmt.Printf("  %s%d.%s %s%s%s%s%s  %s\n",
			colorDim, i+1, colorReset,
			colorYellow, c.Hash, colorReset,
			fmtDate(c.Date, showDate),
			fmtAuthor(c.Author, showAuthor),
			c.Message,
		)
	}
	fmt.Println()
}
