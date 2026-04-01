package main

import (
	"regexp"
	"sort"
	"strings"
)

type TicketEntry struct {
	Ticket  string
	Count   int
	Authors []string
}

func buildTicketSummary(commits []Commit, prefixes []string) map[string][]*TicketEntry {
	patterns := make(map[string]*regexp.Regexp, len(prefixes))
	for _, p := range prefixes {
		patterns[p] = regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(p) + `-\d+\b`)
	}

	entries := make(map[string]*TicketEntry)
	authorSets := make(map[string]map[string]struct{})

	for _, c := range commits {
		seen := make(map[string]struct{})
		for _, rx := range patterns {
			for _, match := range rx.FindAllString(c.Message, -1) {
				ticket := strings.ToUpper(match)
				if _, already := seen[ticket]; already {
					continue
				}
				seen[ticket] = struct{}{}

				if _, ok := entries[ticket]; !ok {
					entries[ticket] = &TicketEntry{Ticket: ticket}
					authorSets[ticket] = make(map[string]struct{})
				}
				entries[ticket].Count++
				if c.Author != "" {
					authorSets[ticket][c.Author] = struct{}{}
				}
			}
		}
	}

	for ticket, authors := range authorSets {
		for a := range authors {
			entries[ticket].Authors = append(entries[ticket].Authors, a)
		}
		sort.Strings(entries[ticket].Authors)
	}

	grouped := make(map[string][]*TicketEntry)
	for _, p := range prefixes {
		prefix := strings.ToUpper(p)
		for _, e := range entries {
			if strings.HasPrefix(e.Ticket, prefix+"-") {
				grouped[prefix] = append(grouped[prefix], e)
			}
		}
		sort.Slice(grouped[prefix], func(i, j int) bool {
			return grouped[prefix][i].Ticket < grouped[prefix][j].Ticket
		})
	}

	return grouped
}
