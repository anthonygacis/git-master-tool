package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func runSearch() {
	var (
		flagSource     = flag.String("source", "", "source branch to search in")
		flagTarget     = flag.String("target", "", "target branch to check against")
		flagShowAuthor = flag.String("show-author", "", "show commit author: true|false")
		flagShowDate   = flag.String("show-date", "", "show commit date: true|false")
		flagFormat     = flag.String("format", "", "output format: default|json")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gitmt search [flags] <message> [source target]\n\nArgs:\n  message   Exact commit message to search for\n  source    Source branch (overrides flag)\n  target    Target branch (overrides flag)\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig file: %s (flags take precedence)\n", configFile)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	pattern := args[0]
	cfg := loadConfig()

	if *flagSource != "" {
		cfg.Source = *flagSource
	}
	if *flagTarget != "" {
		cfg.Target = *flagTarget
	}
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

	switch len(args) {
	case 3:
		cfg.Source = args[1]
		cfg.Target = args[2]
	case 2:
		cfg.Source = args[1]
	}

	if cfg.Source == "" || cfg.Target == "" {
		fmt.Fprintf(os.Stderr, colorRed+"error: "+colorReset+"source and target are required\n")
		flag.Usage()
		os.Exit(1)
	}

	if err := checkGitRepo(); err != nil {
		fatalf("not inside a git repository: %v", err)
	}
	if err := validateBranch(cfg.Source); err != nil {
		fatalf("source branch %q not found: %v", cfg.Source, err)
	}
	if err := validateBranch(cfg.Target); err != nil {
		fatalf("target branch %q not found: %v", cfg.Target, err)
	}

	mergeBase, err := getMergeBase(cfg.Source, cfg.Target)
	if err != nil {
		fatalf("could not find common ancestor: %v", err)
	}

	commits, err := compareByMessage(cfg.Source, cfg.Target, mergeBase)
	if err != nil {
		fatalf("could not compare branches: %v", err)
	}

	var matches []Commit
	for _, c := range commits {
		if c.Message == pattern {
			matches = append(matches, c)
		}
	}

	if cfg.Format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(struct {
			Match bool `json:"match"`
		}{Match: len(matches) > 0})
		return
	}

	showAuthor := cfg.ShowAuthor == nil || *cfg.ShowAuthor
	showDate := cfg.ShowDate == nil || *cfg.ShowDate

	printSearchOutput(cfg.Source, cfg.Target, mergeBase, pattern, matches, showAuthor, showDate)
}

func printSearchOutput(source, target, mergeBase, pattern string, matches []Commit, showAuthor, showDate bool) {
	fmt.Printf("\n%s%sSearch%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s  message : %s%q\n", colorDim, colorReset, pattern)
	fmt.Printf("%s  source  : %s%s\n", colorDim, colorReset, source)
	fmt.Printf("%s  target  : %s%s\n", colorDim, colorReset, target)
	fmt.Printf("%s  base    : %s%s\n", colorDim, colorReset, mergeBase)

	if len(matches) == 0 {
		fmt.Printf("\n%sNo commits with message %q found in %s since merge base.%s\n\n", colorDim, pattern, source, colorReset)
		return
	}

	pending := filterCommits(matches, false)
	applied := filterCommits(matches, true)

	fmt.Printf("%s  found   : %s%d match(es)  %s(%d pending, %d applied)%s\n\n",
		colorDim, colorReset, len(matches),
		colorDim, len(pending), len(applied), colorReset,
	)

	for _, c := range matches {
		var statusColor, statusLabel, statusNote string
		if c.Applied {
			statusColor = colorGreen
			statusLabel = "applied"
			statusNote = colorGreen + "  ✓ already in " + target + colorReset
		} else {
			statusColor = colorYellow
			statusLabel = "pending"
			statusNote = colorYellow + "  ✗ not yet in " + target + colorReset
		}
		fmt.Printf("  %s%-7s%s %s%s%s%s%s  %s%s\n",
			statusColor, statusLabel, colorReset,
			statusColor, c.Hash, colorReset,
			fmtDate(c.Date, showDate),
			fmtAuthor(c.Author, showAuthor),
			c.Message,
			statusNote,
		)
	}
	fmt.Println()
}

