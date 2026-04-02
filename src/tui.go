package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

type tuiItem struct {
	main string
	sub  []string
	tag  string
}

func tuiSelect(header []string, items []tuiItem, multi bool) ([]int, bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Fprintf(os.Stderr, "%serror:%s pick requires an interactive terminal\n", colorRed, colorReset)
		os.Exit(1)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fatalf("could not enter raw terminal mode: %v", err)
	}
	defer term.Restore(fd, oldState)

	fmt.Print("\033[?1049h\033[?25l")
	defer fmt.Print("\033[?1049l\033[?25h")

	_, termH, _ := term.GetSize(fd)
	if termH <= 0 {
		termH = 24
	}

	cursor := 0
	offset := 0 // index of first visible item
	picked := make([]bool, len(items))

	var hint string
	if multi {
		hint = colorDim + "  ↑↓ move   [space] toggle   [enter] confirm   [q] quit" + colorReset
	} else {
		hint = colorDim + "  ↑↓ move   [enter] select   [q] quit" + colorReset
	}

	itemH := func(i int) int { return 1 + len(items[i].sub) }

	// How many lines the fixed chrome uses (header + blank + above-indicator + below-indicator + blank + hint).
	chrome := len(header) + 1 + 1 + 1 + 1 + 1

	// Returns the index of the last item that fits in the viewport from 'from'.
	lastVisible := func(from int) int {
		avail := termH - chrome
		if avail < 1 {
			avail = 1
		}
		last := from
		used := itemH(from)
		for i := from + 1; i < len(items); i++ {
			h := itemH(i)
			if used+h > avail {
				break
			}
			last = i
			used += h
		}
		return last
	}

	// Slide the viewport so the cursor is always visible.
	adjustOffset := func() {
		if cursor < offset {
			offset = cursor
			return
		}
		for cursor > lastVisible(offset) {
			offset++
		}
	}

	draw := func() {
		adjustOffset()
		last := lastVisible(offset)

		fmt.Print("\033[H")
		line := func(s string) { fmt.Printf("\033[2K%s\r\n", s) }

		for _, h := range header {
			line(h)
		}
		line("")

		if offset > 0 {
			line(fmt.Sprintf(colorDim+"  ↑ %d more above"+colorReset, offset))
		} else {
			line("")
		}

		for i := offset; i <= last; i++ {
			it := items[i]

			arrow := "   "
			if i == cursor {
				arrow = colorBold + " ▶ " + colorReset
			}

			check := ""
			if multi {
				if picked[i] {
					check = colorGreen + "[✓] " + colorReset
				} else {
					check = colorDim + "[ ] " + colorReset
				}
			}

			tag := ""
			if it.tag != "" {
				tag = "  " + it.tag
			}

			line(arrow + check + it.main + tag)
			for _, s := range it.sub {
				line("        " + colorDim + s + colorReset)
			}
		}

		below := len(items) - 1 - last
		if below > 0 {
			line(fmt.Sprintf(colorDim+"  ↓ %d more below"+colorReset, below))
		} else {
			line("")
		}

		line("")
		line(hint)
		fmt.Print("\033[0J")
	}

	draw()

	buf := make([]byte, 4)
	for {
		n, _ := os.Stdin.Read(buf)
		if n == 0 {
			continue
		}

		switch {
		case buf[0] == 3 || buf[0] == 'q':
			return nil, false

		case buf[0] == 27 && n == 1:
			return nil, false

		case buf[0] == 13 || buf[0] == 10:
			if !multi {
				return []int{cursor}, true
			}
			var sel []int
			for i, p := range picked {
				if p {
					sel = append(sel, i)
				}
			}
			return sel, len(sel) > 0

		case buf[0] == ' ':
			if multi {
				picked[cursor] = !picked[cursor]
			} else {
				return []int{cursor}, true
			}
			draw()

		case n >= 3 && buf[0] == 27 && buf[1] == '[':
			switch buf[2] {
			case 'A':
				if cursor > 0 {
					cursor--
					draw()
				}
			case 'B':
				if cursor < len(items)-1 {
					cursor++
					draw()
				}
			}
		}
	}
}
