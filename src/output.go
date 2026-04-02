package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func printOutput(cfg Config, source, target, mergeBase string, commits []Commit) {
	showApplied := cfg.Show != "pending"
	showAuthor := cfg.ShowAuthor == nil || *cfg.ShowAuthor
	showDate := cfg.ShowDate == nil || *cfg.ShowDate
	showTicketAuthors := cfg.ShowTicketAuthors == nil || *cfg.ShowTicketAuthors

	var ticketGroups map[string][]*TicketEntry
	if len(cfg.Prefixes) > 0 {
		ticketCommits := commits
		if !showApplied {
			ticketCommits = filterCommits(commits, false)
		}
		ticketGroups = buildTicketSummary(ticketCommits, cfg.Prefixes)
	}

	switch cfg.Format {
	case "table":
		printTable(source, target, mergeBase, commits, showApplied, showAuthor, showDate)
		if len(ticketGroups) > 0 {
			printTicketSummaryTable(ticketGroups, cfg.Prefixes, showTicketAuthors)
		}
	case "json":
		printJSON(source, target, mergeBase, commits, showApplied, showDate, ticketGroups, cfg.Prefixes, showTicketAuthors)
	case "csv":
		printCSV(commits, showApplied, showAuthor, showDate)
	default:
		printDefault(source, target, mergeBase, commits, showApplied, showAuthor, showDate, cfg.Show)
		if len(ticketGroups) > 0 {
			printTicketSummaryDefault(ticketGroups, cfg.Prefixes, showTicketAuthors)
		}
	}
}

func printDefault(source, target, mergeBase string, commits []Commit, showApplied, showAuthor, showDate bool, show string) {
	fmt.Printf("\n%s%sBranch comparison%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s  source : %s%s\n", colorDim, colorReset, source)
	fmt.Printf("%s  target : %s%s\n", colorDim, colorReset, target)
	fmt.Printf("%s  base   : %s%s\n", colorDim, colorReset, mergeBase)
	if show == "pending" {
		fmt.Printf("%s  show   : %spending only%s\n", colorDim, colorReset, colorReset)
	}
	if !showAuthor {
		fmt.Printf("%s  author : %shidden%s\n", colorDim, colorReset, colorReset)
	}
	if !showDate {
		fmt.Printf("%s  date   : %shidden%s\n", colorDim, colorReset, colorReset)
	}

	if len(commits) == 0 {
		fmt.Printf("\n%sNo commits found in %s beyond the common ancestor.%s\n", colorDim, source, colorReset)
		return
	}

	pending := filterCommits(commits, false)
	applied := filterCommits(commits, true)

	if len(pending) > 0 {
		fmt.Printf("\n%s%s Pending — not yet in %s:%s\n", colorBold, colorYellow, target, colorReset)
		for i, c := range pending {
			fmt.Printf("  %s%d.%s %s%s%s%s%s  %s\n",
				colorDim, i+1, colorReset,
				colorYellow, c.Hash, colorReset,
				fmtDate(c.Date, showDate),
				fmtAuthor(c.Author, showAuthor),
				c.Message,
			)
		}
	}

	if showApplied && len(applied) > 0 {
		fmt.Printf("\n%s%s Already applied — commit message found in %s:%s\n", colorBold, colorGreen, target, colorReset)
		for i, c := range applied {
			fmt.Printf("  %s%d.%s %s%s%s%s%s  %s%s%s\n",
				colorDim, i+1, colorReset,
				colorGreen, c.Hash, colorReset,
				fmtDate(c.Date, showDate),
				fmtAuthor(c.Author, showAuthor),
				colorDim, c.Message, colorReset,
			)
		}
	}

	fmt.Printf("\n%s%sSummary%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("  %s%-10s%s %d commit(s) need cherry-picking into %s\n",
		colorYellow, "pending", colorReset, len(pending), target)
	if showApplied {
		fmt.Printf("  %s%-10s%s %d commit(s) already applied\n",
			colorGreen, "applied", colorReset, len(applied))
	}
	fmt.Println()
}

func fmtAuthor(author string, show bool) string {
	if !show || author == "" {
		return ""
	}
	return fmt.Sprintf("  %s(%s)%s", colorDim, author, colorReset)
}

func fmtDate(date string, show bool) string {
	if !show || date == "" {
		return ""
	}
	return fmt.Sprintf("  %s%s%s", colorDim, date, colorReset)
}

func printTable(source, target, mergeBase string, commits []Commit, showApplied, showAuthor, showDate bool) {
	type row struct{ status, hash, date, author, message string }

	var rows []row
	for _, c := range commits {
		if !showApplied && c.Applied {
			continue
		}
		status := "pending"
		if c.Applied {
			status = "applied"
		}
		rows = append(rows, row{status, c.Hash, c.Date, c.Author, c.Message})
	}

	wStatus, wHash, wDate, wAuthor, wMsg := 6, 4, 4, 6, 7
	for _, r := range rows {
		if l := len(r.status); l > wStatus {
			wStatus = l
		}
		if l := len(r.hash); l > wHash {
			wHash = l
		}
		if l := len(r.date); l > wDate {
			wDate = l
		}
		if l := len(r.author); l > wAuthor {
			wAuthor = l
		}
		if l := len(r.message); l > wMsg {
			wMsg = l
		}
	}

	widths := []int{wStatus, wHash, wMsg}
	if showDate && showAuthor {
		widths = []int{wStatus, wHash, wDate, wAuthor, wMsg}
	} else if showDate {
		widths = []int{wStatus, wHash, wDate, wMsg}
	} else if showAuthor {
		widths = []int{wStatus, wHash, wAuthor, wMsg}
	}

	border := func(left, mid, right, fill string) string {
		var parts []string
		for _, w := range widths {
			parts = append(parts, strings.Repeat(fill, w+2))
		}
		return left + strings.Join(parts, mid) + right
	}

	cell := func(w int, val, color string) string {
		padded := fmt.Sprintf("%-*s", w, val)
		if color != "" {
			return " " + color + padded + colorReset + " "
		}
		return " " + padded + " "
	}

	rowLine := func(r row) string {
		statusColor := colorYellow
		if r.status == "applied" {
			statusColor = colorGreen
		}
		cells := []string{
			cell(wStatus, r.status, statusColor),
			cell(wHash, r.hash, colorDim),
		}
		if showDate {
			cells = append(cells, cell(wDate, r.date, colorDim))
		}
		if showAuthor {
			cells = append(cells, cell(wAuthor, r.author, ""))
		}
		cells = append(cells, cell(wMsg, r.message, ""))
		return "│" + strings.Join(cells, "│") + "│"
	}

	headerLine := func() string {
		cells := []string{
			cell(wStatus, "STATUS", colorBold),
			cell(wHash, "HASH", colorBold),
		}
		if showDate {
			cells = append(cells, cell(wDate, "DATE", colorBold))
		}
		if showAuthor {
			cells = append(cells, cell(wAuthor, "AUTHOR", colorBold))
		}
		cells = append(cells, cell(wMsg, "MESSAGE", colorBold))
		return "│" + strings.Join(cells, "│") + "│"
	}

	fmt.Printf("\n%s%sBranch comparison%s  %s → %s  %sbase: %s%s\n\n",
		colorBold, colorCyan, colorReset,
		source, target,
		colorDim, mergeBase, colorReset,
	)

	fmt.Println(border("┌", "┬", "┐", "─"))
	fmt.Println(headerLine())
	fmt.Println(border("├", "┼", "┤", "─"))

	if len(rows) == 0 {
		msg := "no commits found"
		totalWidth := 0
		for _, w := range widths {
			totalWidth += w + 3
		}
		totalWidth -= 1
		fmt.Printf("│ %-*s │\n", totalWidth, msg)
	} else {
		for _, r := range rows {
			fmt.Println(rowLine(r))
		}
	}

	fmt.Println(border("└", "┴", "┘", "─"))

	pending := len(filterCommits(commits, false))
	applied := len(filterCommits(commits, true))
	fmt.Printf("\n  %s%d pending%s", colorYellow, pending, colorReset)
	if showApplied {
		fmt.Printf("   %s%d applied%s", colorGreen, applied, colorReset)
	}
	fmt.Println()
}

type jsonOutput struct {
	Source        string           `json:"source"`
	Target        string           `json:"target"`
	Base          string           `json:"base"`
	Commits       []jsonCommit     `json:"commits"`
	Summary       jsonSummary      `json:"summary"`
	TicketSummary []jsonTicketGroup `json:"ticket_summary,omitempty"`
}

type jsonCommit struct {
	Hash    string `json:"hash"`
	Date    string `json:"date,omitempty"`
	Author  string `json:"author"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

type jsonSummary struct {
	Pending int `json:"pending"`
	Applied int `json:"applied"`
}

type jsonTicketGroup struct {
	Prefix  string            `json:"prefix"`
	Total   int               `json:"total"`
	Tickets []jsonTicketEntry `json:"tickets"`
}

type jsonTicketEntry struct {
	Ticket  string   `json:"ticket"`
	Count   int      `json:"count"`
	Authors []string `json:"authors,omitempty"`
}

func printJSON(source, target, mergeBase string, commits []Commit, showApplied, showDate bool, ticketGroups map[string][]*TicketEntry, prefixes []string, showTicketAuthors bool) {
	jCommits := make([]jsonCommit, 0, len(commits))
	pending, applied := 0, 0

	for _, c := range commits {
		status := "pending"
		if c.Applied {
			status = "applied"
			applied++
		} else {
			pending++
		}
		if !showApplied && c.Applied {
			continue
		}
		jc := jsonCommit{
			Hash:    c.Hash,
			Author:  c.Author,
			Message: c.Message,
			Status:  status,
		}
		if showDate {
			jc.Date = c.Date
		}
		jCommits = append(jCommits, jc)
	}

	var jTickets []jsonTicketGroup
	for _, p := range prefixes {
		prefix := strings.ToUpper(p)
		entries := ticketGroups[prefix]
		jEntries := make([]jsonTicketEntry, 0, len(entries))
		total := 0
		for _, e := range entries {
			je := jsonTicketEntry{Ticket: e.Ticket, Count: e.Count}
			if showTicketAuthors {
				je.Authors = e.Authors
			}
			jEntries = append(jEntries, je)
			total += e.Count
		}
		jTickets = append(jTickets, jsonTicketGroup{Prefix: prefix, Total: total, Tickets: jEntries})
	}

	out := jsonOutput{
		Source:        source,
		Target:        target,
		Base:          mergeBase,
		Commits:       jCommits,
		Summary:       jsonSummary{Pending: pending, Applied: applied},
		TicketSummary: jTickets,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func printTicketSummaryDefault(groups map[string][]*TicketEntry, prefixes []string, showAuthors bool) {
	fmt.Printf("\n%s%sTicket Summary%s\n", colorBold, colorCyan, colorReset)
	for _, p := range prefixes {
		prefix := strings.ToUpper(p)
		entries := groups[prefix]
		if len(entries) == 0 {
			continue
		}
		total := 0
		fmt.Printf("\n%s%s%s\n", colorBold, prefix, colorReset)
		for _, e := range entries {
			commits := "commit"
			if e.Count > 1 {
				commits = "commits"
			}
			fmt.Printf("  %s%-12s%s %s%d %s%s",
				colorYellow, e.Ticket, colorReset,
				colorDim, e.Count, commits, colorReset,
			)
			if showAuthors && len(e.Authors) > 0 {
				fmt.Printf("   %s%s%s", colorDim, strings.Join(e.Authors, ", "), colorReset)
			}
			fmt.Println()
			total += e.Count
		}
		totalCommits := "commit"
		if total > 1 {
			totalCommits = "commits"
		}
		fmt.Printf("  %s%-12s%s %s%d %s%s\n", colorBold, "total", colorReset, colorBold, total, totalCommits, colorReset)
	}
	fmt.Println()
}

func printTicketSummaryTable(groups map[string][]*TicketEntry, prefixes []string, showAuthors bool) {
	fmt.Printf("\n%s%sTicket Summary%s\n", colorBold, colorCyan, colorReset)

	for _, p := range prefixes {
		prefix := strings.ToUpper(p)
		entries := groups[prefix]
		if len(entries) == 0 {
			continue
		}

		wTicket, wCount, wAuthors := 6, 6, 7
		for _, e := range entries {
			if l := len(e.Ticket); l > wTicket {
				wTicket = l
			}
			label := fmt.Sprintf("%d commit(s)", e.Count)
			if l := len(label); l > wCount {
				wCount = l
			}
			if showAuthors {
				if l := len(strings.Join(e.Authors, ", ")); l > wAuthors {
					wAuthors = l
				}
			}
		}

		widths := []int{wTicket, wCount}
		if showAuthors {
			widths = append(widths, wAuthors)
		}

		border := func(left, mid, right, fill string) string {
			var parts []string
			for _, w := range widths {
				parts = append(parts, strings.Repeat(fill, w+2))
			}
			return left + strings.Join(parts, mid) + right
		}

		cell := func(w int, val, color string) string {
			padded := fmt.Sprintf("%-*s", w, val)
			if color != "" {
				return " " + color + padded + colorReset + " "
			}
			return " " + padded + " "
		}

		fmt.Printf("\n%s%s%s\n", colorBold, prefix, colorReset)
		fmt.Println(border("┌", "┬", "┐", "─"))

		header := "│" + cell(wTicket, "TICKET", colorBold) + "│" + cell(wCount, "COMMITS", colorBold)
		if showAuthors {
			header += "│" + cell(wAuthors, "AUTHORS", colorBold)
		}
		fmt.Println(header + "│")
		fmt.Println(border("├", "┼", "┤", "─"))

		total := 0
		for _, e := range entries {
			row := "│" +
				cell(wTicket, e.Ticket, colorYellow) + "│" +
				cell(wCount, fmt.Sprintf("%d commit(s)", e.Count), "")
			if showAuthors {
				row += "│" + cell(wAuthors, strings.Join(e.Authors, ", "), colorDim)
			}
			fmt.Println(row + "│")
			total += e.Count
		}
		fmt.Println(border("├", "┼", "┤", "─"))
		totalRow := "│" +
			cell(wTicket, "total", colorBold) + "│" +
			cell(wCount, fmt.Sprintf("%d commit(s)", total), colorBold)
		if showAuthors {
			totalRow += "│" + cell(wAuthors, "", "")
		}
		fmt.Println(totalRow + "│")
		fmt.Println(border("└", "┴", "┘", "─"))
	}
	fmt.Println()
}

func printCSV(commits []Commit, showApplied, showAuthor, showDate bool) {
	w := csv.NewWriter(os.Stdout)

	header := []string{"status", "hash"}
	if showDate {
		header = append(header, "date")
	}
	if showAuthor {
		header = append(header, "author")
	}
	header = append(header, "message")
	_ = w.Write(header)

	for _, c := range commits {
		if !showApplied && c.Applied {
			continue
		}
		status := "pending"
		if c.Applied {
			status = "applied"
		}
		record := []string{status, c.Hash}
		if showDate {
			record = append(record, c.Date)
		}
		if showAuthor {
			record = append(record, c.Author)
		}
		record = append(record, c.Message)
		_ = w.Write(record)
	}

	w.Flush()
}
