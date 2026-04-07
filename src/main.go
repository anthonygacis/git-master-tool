package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const configFile = ".gitmt.yml"

var version = "dev"

type Config struct {
	Source            string   `yaml:"source"`
	Target            string   `yaml:"target"`
	Show              string   `yaml:"show"`
	ShowAuthor        *bool    `yaml:"show_author"`
	ShowDate          *bool    `yaml:"show_date"`
	Format            string   `yaml:"format"`
	Prefixes          []string `yaml:"prefixes"`
	ShowTicketAuthors *bool    `yaml:"show_ticket_authors"`
}

func main() {
	if len(os.Args) < 2 {
		printMainUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)

	switch cmd {
	case "compare":
		runCompare()
	case "scan":
		runScan()
	case "pick":
		runPick()
	case "search":
		runSearch()
	case "tagdiff":
		runTagDiff()
	case "upgrade":
		runUpgrade()
	case "version", "--version", "-version":
		fmt.Printf("gitmt %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", cmd)
		printMainUsage()
		os.Exit(1)
	}
}

func printMainUsage() {
	fmt.Fprintf(os.Stderr, "Usage: gitmt <command> [flags]\n\nCommands:\n  compare   Compare two git branches\n  scan      Group pending commits into safe cherry-pick batches\n  pick      Interactively cherry-pick a batch or individual commit\n  search    Search for a commit message across branches\n  tagdiff   List commits between two tags\n  upgrade   Upgrade gitmt to the latest release\n  version   Print version\n\nRun 'gitmt <command> --help' for more information.\n")
}

func runCompare() {
	var (
		flagSource            = flag.String("source", "", "source branch")
		flagTarget            = flag.String("target", "", "target branch")
		flagShow              = flag.String("show", "", "output filter: pending|all")
		flagShowAuthor        = flag.String("show-author", "", "show commit author: true|false")
		flagShowDate          = flag.String("show-date", "", "show commit date: true|false")
		flagFormat            = flag.String("format", "", "output format: default|table|json|csv")
		flagPrefixes          = flag.String("prefixes", "", "jira prefixes for ticket summary, comma-separated (e.g. XS,XI)")
		flagShowTicketAuthors = flag.String("show-ticket-authors", "", "show authors in ticket summary: true|false")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gitmt compare [flags] [source target]\n\nFlags:\n")
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
	if *flagShow != "" {
		cfg.Show = *flagShow
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
	if *flagPrefixes != "" {
		cfg.Prefixes = splitPrefixes(*flagPrefixes)
	}
	if *flagShowTicketAuthors != "" {
		v := *flagShowTicketAuthors == "true"
		cfg.ShowTicketAuthors = &v
	}

	if args := flag.Args(); len(args) == 2 {
		cfg.Source = args[0]
		cfg.Target = args[1]
	} else if len(args) != 0 {
		flag.Usage()
		os.Exit(1)
	}

	if cfg.Source == "" || cfg.Target == "" {
		fmt.Fprintf(os.Stderr, colorRed+"error: "+colorReset+"source and target are required (flags, positional args, or %s)\n", configFile)
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

	printOutput(cfg, cfg.Source, cfg.Target, mergeBase, commits)
}

func loadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg
		}
		fatalf("reading %s: %v", configFile, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fatalf("parsing %s: %v", configFile, err)
	}
	fmt.Fprintf(os.Stderr, "%susing config: %s%s\n", colorDim, configFile, colorReset)
	return cfg
}

func splitPrefixes(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorRed+"error: "+colorReset+format+"\n", args...)
	os.Exit(1)
}
