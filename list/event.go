// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package list

import (
	"os"
	"strings"

	"github.com/nsf/termbox-go"
)

// InsertRune adds rune to search keywords(l.Keyword).
func (l *Info) InsertRune(inputRune rune) {
	l.Keyword += string(inputRune)
}

// DeleteRune deletes rune at search keywords(l.Keyword).
func (l *Info) DeleteRune() {
	sc := []rune(l.Keyword)
	l.Keyword = string(sc[:(len(sc) - 1)])
}

// keyEvent waits for keyboard events.
func (l *Info) keyEvent() {
	l.CursorLine = 0
	headLine := 2

	_, height := termbox.Size()
	height -= headLine

	l.Keyword = ""
	allFlag := false // input Ctrl + A flag

	l.getFilterText()
	l.draw()

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		// Type Key
		case termbox.EventKey:
			switch ev.Key {
			// ESC or Ctrl + C Key (Exit)
			case termbox.KeyEsc, termbox.KeyCtrlC:
				termbox.Close()
				os.Exit(0)

			// AllowUp Key
			case termbox.KeyArrowUp:
				if l.CursorLine > 0 {
					l.CursorLine--
				} else { // 掉头到最低行
					l.CursorLine = len(l.ViewText) - headLine
				}

				l.draw()

			// AllowDown Key
			case termbox.KeyArrowDown:
				if l.CursorLine < len(l.ViewText)-headLine {
					l.CursorLine++
				} else { // 掉头到第一行
					l.CursorLine = 0
				}

				l.draw()

			// AllowRight Key
			case termbox.KeyArrowRight:
				nextPosition := ((l.CursorLine + height) / height) * height
				if nextPosition+2 <= len(l.ViewText) {
					l.CursorLine = nextPosition
				}

				l.draw()

			// AllowLeft Key
			case termbox.KeyArrowLeft:
				beforePosition := ((l.CursorLine - height) / height) * height
				if beforePosition >= 0 {
					l.CursorLine = beforePosition
				}

				l.draw()

			// Tab Key(select)
			case termbox.KeyTab:
				if l.MultiFlag {
					l.toggle(strings.Fields(l.ViewText[l.CursorLine+1])[0])
				}

				if l.CursorLine < len(l.ViewText)-headLine {
					l.CursorLine++
				}

				l.draw()

			// Ctrl + a Key(all select)
			case termbox.KeyCtrlA:
				if l.MultiFlag {
					l.allToggle(allFlag)
					// allFlag Toggle
					allFlag = !allFlag
				}

				l.draw()

			// Ctrl + h Key(Help Window)
			// case termbox.KeyCtrlH:

			// Enter Key
			case termbox.KeyEnter:
				if len(l.SelectName) == 0 {
					l.SelectName = append(l.SelectName, strings.Fields(l.ViewText[l.CursorLine+1])[0])
				}

				return

			// BackSpace Key
			case termbox.KeyBackspace, termbox.KeyBackspace2:
				if len(l.Keyword) > 0 {
					l.DeleteRune()
					l.getFilterText()

					if l.CursorLine > len(l.ViewText) {
						l.CursorLine = len(l.ViewText)
					}

					if l.CursorLine < 0 {
						l.CursorLine = 0
					}

					allFlag = false

					l.draw()
				}

			// Space Key
			case termbox.KeySpace:
				l.Keyword += " "
				l.draw()

			// Other Key
			default:
				if ev.Ch != 0 {
					l.InsertRune(ev.Ch)
					l.getFilterText()

					if l.CursorLine > len(l.ViewText)-headLine {
						l.CursorLine = len(l.ViewText) - headLine
					}

					allFlag = false

					l.draw()
				}
			}

		// Type Mouse
		case termbox.EventMouse:
			if ev.Key == termbox.MouseLeft {
				// mouse select line is (ev.MouseY - headLine) line.
				mouseSelectLine := ev.MouseY - headLine

				if mouseSelectLine <= len(l.ViewText)-headLine {
					l.CursorLine = mouseSelectLine
				}

				l.draw()
			}

		// Other
		default:
			l.draw()
		}
	}
}
