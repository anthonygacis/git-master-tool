package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func runScan() {
	var (
		flagSource     = flag.String("source", "", "source branch")
		flagTarget     = flag.String("target", "", "target branch")
		flagShowAuthor = flag.String("show-author", "", "show commit author: true|false")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gitmt scan [flags] [source target]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig file: %s (flags take precedence)\n", configFile)
	}
	flag.Parse()

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
	if args := flag.Args(); len(args) == 2 {
		cfg.Source = args[0]
		cfg.Target = args[1]
	} else if len(args) != 0 {
		flag.Usage()
		os.Exit(1)
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

	pending := filterCommits(commits, false)
	if len(pending) == 0 {
		fmt.Printf("\n%sAll commits already applied — nothing to scan.%s\n\n", colorGreen, colorReset)
		return
	}

	commitFiles := make(map[string][]string, len(pending))
	for _, c := range pending {
		files, err := getCommitFiles(c.Hash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%swarn:%s could not get files for %s: %v\n", colorYellow, colorReset, c.Hash, err)
		}
		commitFiles[c.Hash] = files
	}

	batches := groupBySharedFiles(pending, commitFiles)
	showAuthor := cfg.ShowAuthor == nil || *cfg.ShowAuthor
	printScanOutput(cfg.Source, cfg.Target, batches, commitFiles, showAuthor)
}

type scanBatch struct {
	commits     []Commit
	sharedFiles []string
}

func groupBySharedFiles(commits []Commit, commitFiles map[string][]string) []scanBatch {
	n := len(commits)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y int) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	fileToIdx := make(map[string][]int)
	for i, c := range commits {
		for _, f := range commitFiles[c.Hash] {
			fileToIdx[f] = append(fileToIdx[f], i)
		}
	}
	for _, indices := range fileToIdx {
		for i := 1; i < len(indices); i++ {
			union(indices[0], indices[i])
		}
	}

	type group struct {
		minIdx  int
		indices []int
	}
	groups := make(map[int]*group)
	for i := range commits {
		root := find(i)
		g := groups[root]
		if g == nil {
			g = &group{minIdx: i}
			groups[root] = g
		}
		if i < g.minIdx {
			g.minIdx = i
		}
		g.indices = append(g.indices, i)
	}

	roots := make([]int, 0, len(groups))
	for root := range groups {
		roots = append(roots, root)
	}
	sort.Slice(roots, func(a, b int) bool {
		return groups[roots[a]].minIdx < groups[roots[b]].minIdx
	})

	batches := make([]scanBatch, 0, len(roots))
	for _, root := range roots {
		g := groups[root]
		sort.Ints(g.indices)

		bCommits := make([]Commit, len(g.indices))
		for i, idx := range g.indices {
			bCommits[i] = commits[idx]
		}

		var sharedFiles []string
		if len(bCommits) > 1 {
			fileCounts := make(map[string]int)
			for _, c := range bCommits {
				for _, f := range commitFiles[c.Hash] {
					fileCounts[f]++
				}
			}
			for f, cnt := range fileCounts {
				if cnt > 1 {
					sharedFiles = append(sharedFiles, f)
				}
			}
			sort.Strings(sharedFiles)
		}

		batches = append(batches, scanBatch{commits: bCommits, sharedFiles: sharedFiles})
	}
	return batches
}

func printScanOutput(source, target string, batches []scanBatch, commitFiles map[string][]string, showAuthor bool) {
	totalPending := 0
	for _, b := range batches {
		totalPending += len(b.commits)
	}

	fmt.Printf("\n%s%sScan — cherry-pick batches%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s  source  : %s%s\n", colorDim, colorReset, source)
	fmt.Printf("%s  target  : %s%s\n", colorDim, colorReset, target)
	fmt.Printf("%s  pending : %s%d commit(s)\n", colorDim, colorReset, totalPending)
	fmt.Printf("%s  batches : %s%d\n\n", colorDim, colorReset, len(batches))

	for i, b := range batches {
		if len(b.commits) == 1 {
			fmt.Printf("%s● Batch %d%s  %s1 commit — independent%s\n",
				colorGreen, i+1, colorReset, colorDim, colorReset)
		} else {
			fmt.Printf("%s● Batch %d%s  %s%d commits — cherry-pick together%s\n",
				colorYellow, i+1, colorReset, colorBold, len(b.commits), colorReset)
		}

		for _, c := range b.commits {
			nFiles := len(commitFiles[c.Hash])
			fmt.Printf("  %s%s%s%s  %s  %s(%d file(s))%s\n",
				colorDim, c.Hash, colorReset,
				fmtAuthor(c.Author, showAuthor),
				c.Message,
				colorDim, nFiles, colorReset,
			)
		}

		if len(b.sharedFiles) > 0 {
			fmt.Printf("  %sshared:%s\n", colorDim, colorReset)
			for _, f := range b.sharedFiles {
				fmt.Printf("    %s%s%s\n", colorDim, f, colorReset)
			}
		}
		fmt.Println()
	}
}
