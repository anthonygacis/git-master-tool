package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runPick() {
	var (
		flagSource     = flag.String("source", "", "source branch")
		flagTarget     = flag.String("target", "", "target branch")
		flagShowAuthor = flag.String("show-author", "", "show commit author: true|false")
		flagShowDate   = flag.String("show-date", "", "show commit date: true|false")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gitmt pick [flags] [source target]\n\nFlags:\n")
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
	if *flagShowDate != "" {
		v := *flagShowDate == "true"
		cfg.ShowDate = &v
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
		fmt.Printf("\n%sAll commits already applied — nothing to cherry-pick.%s\n\n", colorGreen, colorReset)
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
	showDate := cfg.ShowDate == nil || *cfg.ShowDate

	mode := pickMode(cfg.Source, cfg.Target, len(pending), len(batches))
	if mode < 0 {
		return
	}

	var hashes []string
	switch mode {
	case 0:
		hashes = pickBatches(batches, pending, showAuthor, showDate)
	case 1:
		hashes = pickIndividual(batches, pending, showAuthor, showDate)
	}

	if len(hashes) == 0 {
		fmt.Printf("%sNothing selected.%s\n", colorDim, colorReset)
		return
	}

	if !confirmPick(cfg.Target, hashes) {
		fmt.Printf("%sCancelled.%s\n\n", colorDim, colorReset)
		return
	}

	executeCherryPick(cfg.Target, hashes)
}

func pickMode(source, target string, nPending, nBatches int) int {
	items := []tuiItem{
		{
			main: colorBold + "Batch" + colorReset,
			tag:  colorDim + "cherry-pick related commits together" + colorReset,
		},
		{
			main: colorBold + "Individual" + colorReset,
			tag:  colorDim + "pick any commits regardless of grouping" + colorReset,
		},
	}
	header := []string{
		fmt.Sprintf("%s%sCherry-pick%s   %s%s → %s%s", colorBold, colorCyan, colorReset, colorDim, source, target, colorReset),
		fmt.Sprintf("%s  %d pending commit(s)   %d batch(es)%s", colorDim, nPending, nBatches, colorReset),
	}
	sel, ok := tuiSelect(header, items, false, nil)
	if !ok {
		return -1
	}
	return sel[0]
}

func pickBatches(batches []scanBatch, pending []Commit, showAuthor, showDate bool) []string {
	items := make([]tuiItem, len(batches))
	for i, b := range batches {
		var main string
		if len(b.commits) == 1 {
			main = fmt.Sprintf("%sBatch %d%s  1 commit — independent", colorBold, i+1, colorReset)
		} else {
			main = fmt.Sprintf("%s%sBatch %d%s  %d commits — cherry-pick together", colorYellow, colorBold, i+1, colorReset, len(b.commits))
		}

		var sub []string
		for _, c := range b.commits {
			author := ""
			if showAuthor && c.Author != "" {
				author = "  (" + c.Author + ")"
			}
			date := ""
			if showDate && c.Date != "" {
				date = "  " + c.Date
			}
			sub = append(sub, c.Hash+date+author+"  "+c.Message)
		}
		if len(b.sharedFiles) > 0 {
			sub = append(sub, "shared: "+strings.Join(b.sharedFiles, ", "))
		}

		items[i] = tuiItem{main: main, sub: sub}
	}

	header := []string{
		colorBold + colorCyan + "Select batches to cherry-pick" + colorReset,
		colorDim + fmt.Sprintf("  %d batch(es)", len(batches)) + colorReset,
	}

	batchStatusFn := func(picked []bool) string {
		var nums []string
		total := 0
		for i, p := range picked {
			if p {
				nums = append(nums, fmt.Sprintf("%d", i+1))
				total += len(batches[i].commits)
			}
		}
		if len(nums) == 0 {
			return colorDim + "  Selected: none" + colorReset
		}
		noun := "commit"
		if total != 1 {
			noun = "commits"
		}
		return fmt.Sprintf("%s  Selected: Batch %s%s  %s(%d %s)%s",
			colorBold, strings.Join(nums, ", "), colorReset,
			colorDim, total, noun, colorReset,
		)
	}

	sel, ok := tuiSelect(header, items, true, batchStatusFn)
	if !ok || len(sel) == 0 {
		return nil
	}

	selected := make(map[string]bool)
	for _, idx := range sel {
		for _, c := range batches[idx].commits {
			selected[c.Hash] = true
		}
	}
	return chronologicalHashes(pending, selected)
}

func pickIndividual(batches []scanBatch, pending []Commit, showAuthor, showDate bool) []string {
	batchOf := make(map[string]int)
	for i, b := range batches {
		if len(b.commits) > 1 {
			for _, c := range b.commits {
				batchOf[c.Hash] = i + 1
			}
		}
	}

	items := make([]tuiItem, len(pending))
	for i, c := range pending {
		author := ""
		if showAuthor && c.Author != "" {
			author = "  (" + c.Author + ")"
		}
		date := ""
		if showDate && c.Date != "" {
			date = "  " + c.Date
		}
		tag := ""
		if bn, inBatch := batchOf[c.Hash]; inBatch {
			tag = colorYellow + fmt.Sprintf("⚠ batch %d", bn) + colorReset
		}
		items[i] = tuiItem{
			main: c.Hash + date + author + "  " + c.Message,
			tag:  tag,
		}
	}

	header := []string{
		colorBold + colorCyan + "Select commits to cherry-pick" + colorReset,
		colorDim + fmt.Sprintf("  %d pending commit(s)", len(pending)) + colorReset,
	}

	sel, ok := tuiSelect(header, items, true, nil)
	if !ok || len(sel) == 0 {
		return nil
	}

	selected := make(map[string]bool)
	for _, idx := range sel {
		selected[pending[idx].Hash] = true
	}
	return chronologicalHashes(pending, selected)
}

func chronologicalHashes(pending []Commit, selected map[string]bool) []string {
	var hashes []string
	for i := len(pending) - 1; i >= 0; i-- {
		if selected[pending[i].Hash] {
			hashes = append(hashes, pending[i].Hash)
		}
	}
	return hashes
}

func confirmPick(target string, hashes []string) bool {
	noun := "1 commit"
	if len(hashes) > 1 {
		noun = fmt.Sprintf("%d commits", len(hashes))
	}
	fmt.Printf("\n%sAbout to cherry-pick %s into %s%s%s:%s\n  git cherry-pick %s\n\n",
		colorDim, noun, colorReset+colorBold, target, colorReset+colorDim, colorReset,
		strings.Join(hashes, " "),
	)
	fmt.Printf("Proceed? %s[y]%ses / %s[n]%so: ", colorBold, colorReset, colorBold, colorReset)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Printf("%sinvalid — enter y or n%s\n", colorRed, colorReset)
			fmt.Printf("Proceed? %s[y]%ses / %s[n]%so: ", colorBold, colorReset, colorBold, colorReset)
		}
	}
	return false
}

func executeCherryPick(target string, hashes []string) {
	current, err := getCurrentBranch()
	if err != nil {
		fatalf("could not determine current branch: %v", err)
	}

	if current != target {
		fmt.Printf("\n%s→%s Switching to %s%s%s\n", colorDim, colorReset, colorBold, target, colorReset)
		checkout := exec.Command("git", "checkout", target)
		checkout.Stdout = os.Stdout
		checkout.Stderr = os.Stderr
		if err := checkout.Run(); err != nil {
			fatalf("could not checkout %s: %v", target, err)
		}
	}

	if cherryPickInProgress() {
		fmt.Printf("%s→%s Aborting previous cherry-pick...\n", colorDim, colorReset)
		abort := exec.Command("git", "cherry-pick", "--abort")
		abort.Stdout = os.Stdout
		abort.Stderr = os.Stderr
		abort.Run()
	}

	args := append([]string{"cherry-pick"}, hashes...)
	fmt.Printf("\n%s→%s git %s\n\n", colorDim, colorReset, strings.Join(args, " "))

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n%serror:%s cherry-pick failed — resolve conflicts then:\n", colorRed, colorReset)
		fmt.Fprintf(os.Stderr, "  %sgit cherry-pick --continue%s  apply resolved commits\n", colorBold, colorReset)
		fmt.Fprintf(os.Stderr, "  %sgit cherry-pick --skip%s     skip the conflicting commit\n", colorBold, colorReset)
		fmt.Fprintf(os.Stderr, "  %sgit cherry-pick --abort%s    cancel and return to original state\n", colorBold, colorReset)
		os.Exit(1)
	}
	fmt.Printf("\n%s✓%s Cherry-pick complete\n\n", colorGreen, colorReset)
}
