package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const configFile = ".gitcompare.yml"

var version = "dev"

type Config struct {
	Source            string   `yaml:"source"`
	Target            string   `yaml:"target"`
	Show              string   `yaml:"show"`
	ShowAuthor        *bool    `yaml:"show_author"`
	Format            string   `yaml:"format"`
	Prefixes          []string `yaml:"prefixes"`
	ShowTicketAuthors *bool    `yaml:"show_ticket_authors"`
}

func main() {
	var (
		flagVersion           = flag.Bool("version", false, "print version and exit")
		flagSource            = flag.String("source", "", "source branch")
		flagTarget            = flag.String("target", "", "target branch")
		flagShow              = flag.String("show", "", "output filter: pending|all")
		flagShowAuthor        = flag.String("show-author", "", "show commit author: true|false")
		flagFormat            = flag.String("format", "", "output format: default|table|json|csv")
		flagPrefixes          = flag.String("prefixes", "", "jira prefixes for ticket summary, comma-separated (e.g. XS,XI)")
		flagShowTicketAuthors = flag.String("show-ticket-authors", "", "show authors in ticket summary: true|false")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [source target]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig file: %s (flags take precedence)\n", configFile)
	}
	flag.Parse()

	if *flagVersion {
		fmt.Printf("gitcompare %s\n", version)
		os.Exit(0)
	}

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
