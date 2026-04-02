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

	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	cursor := 0
	picked := make([]bool, len(items))
	nLines := 0
	first := true

	var hint string
	if multi {
		hint = colorDim + "  ↑↓ move   [space] toggle   [enter] confirm   [q] quit" + colorReset
	} else {
		hint = colorDim + "  ↑↓ move   [enter] select   [q] quit" + colorReset
	}

	draw := func() {
		if !first {
			fmt.Printf("\033[%dA\r\033[0J", nLines)
		}
		first = false
		nLines = 0

		line := func(s string) {
			fmt.Printf("%s\r\n", s)
			nLines++
		}

		for _, h := range header {
			line(h)
		}
		line("")

		for i, it := range items {
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

		line("")
		line(hint)
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
			fmt.Printf("\033[%dA\r\033[0J", nLines)
			return nil, false

		case buf[0] == 27 && n == 1:
			fmt.Printf("\033[%dA\r\033[0J", nLines)
			return nil, false

		case buf[0] == 13 || buf[0] == 10:
			fmt.Printf("\033[%dA\r\033[0J", nLines)
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
				fmt.Printf("\033[%dA\r\033[0J", nLines)
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
