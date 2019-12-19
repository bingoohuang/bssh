// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Package list creates a TUI list based on the contents specified in a structure, and returns the selected row.
package list

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/tabwriter"

	termbox "github.com/nsf/termbox-go"
)

// TODO(blacknon):
//     - 外部のライブラリとして外出しする
//     - tomlやjsonなどを渡して、出力項目を指定できるようにする
//     - 指定した項目だけでの検索などができるようにする
//     - 検索方法の充実化(regexでの検索など)
//     - 内部でのウィンドウの実装
//         - 項目について、更新や閲覧ができるようにする
//     - キーバインドの設定変更

// Info ...
type Info struct {
	// Incremental search line prompt string
	Prompt string

	Title string
	RowFn func(name string) string

	NameList   []string
	SelectName []string
	DataText   []string // all data text list
	ViewText   []string // filtered text list
	MultiFlag  bool     // multi select flag
	Keyword    string   // input keyword
	CursorLine int      // cursor line
	Term       TermInfo
}

// TermInfo ...
type TermInfo struct {
	Headline        int
	LeftMargin      int
	Color           int
	BackgroundColor int
}

// ArrayInfo ...
type ArrayInfo struct {
	Name    string
	Connect string
	Note    string
}

// arrayContains returns that arr contains str.
func arrayContains(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}

	return false
}

// toggle the selected state of cursor line.
func (l *Info) toggle(newLine string) {
	tmpList := make([]string, 0)

	addFlag := true

	for _, selectedLine := range l.SelectName {
		if selectedLine != newLine {
			tmpList = append(tmpList, selectedLine)
		} else {
			addFlag = false
		}
	}

	if addFlag {
		tmpList = append(tmpList, newLine)
	}

	l.SelectName = tmpList
}

// allToggle the selected state of the currently displayed list
func (l *Info) allToggle(allFlag bool) {
	// allFlag is False
	if !allFlag {
		// On each lines that except a header line and are not selected line,
		// toggles left end fields
		for _, addLine := range l.ViewText[1:] {
			addName := strings.Fields(addLine)[0]
			if !arrayContains(l.SelectName, addName) {
				l.toggle(addName)
			}
		}

		return
	}

	// On each lines that except a header line, toggles left end fields
	for _, addLine := range l.ViewText[1:] {
		addName := strings.Fields(addLine)[0]
		l.toggle(addName)
	}
}

// SetTitle sets the view's title columns
func (l *Info) SetTitle(titleColumns []string) {
	s := ""
	for _, col := range titleColumns {
		s += col + "\t"
	}

	l.Title = s
}

// Create view text (use text/tabwriter)
func (l *Info) getText() {
	buffer := &bytes.Buffer{}
	tabWriterBuffer := new(tabwriter.Writer)
	tabWriterBuffer.Init(buffer, 0, 4, 8, ' ', 0)
	fmt.Fprintln(tabWriterBuffer, l.Title)

	// Create list table
	for _, key := range l.NameList {
		fmt.Fprintln(tabWriterBuffer, l.RowFn(key))
	}

	tabWriterBuffer.Flush()

	line, err := buffer.ReadString('\n')

	for err == nil {
		str := strings.Replace(line, "\t", " ", -1)
		l.DataText = append(l.DataText, str)
		line, err = buffer.ReadString('\n')
	}
}

// getFilterText updates l.ViewText with matching keyword (ignore case).
// DataText sets ViewText if keyword is empty.
func (l *Info) getFilterText() {
	// Initialization ViewText
	l.ViewText = []string{}

	// SearchText Bounds Space
	keywords := strings.Fields(l.Keyword)
	r := l.DataText[1:]

	var tmpText []string

	l.ViewText = append(l.ViewText, l.DataText[0])

	// if No words
	if len(keywords) == 0 {
		l.ViewText = l.DataText
		return
	}

	for i := 0; i < len(keywords); i++ {
		lowKeyword := regexp.QuoteMeta(strings.ToLower(keywords[i]))
		re := regexp.MustCompile(lowKeyword)
		tmpText = []string{}

		for j := 0; j < len(r); j++ {
			line := r[j]
			if re.MatchString(strings.ToLower(line)) {
				tmpText = append(tmpText, line)
			}
		}

		r = tmpText
	}

	l.ViewText = append(l.ViewText, tmpText...)
}

// View displays the list in TUI
func (l *Info) View() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}

	defer termbox.Close()

	// enable termbox mouse input
	termbox.SetInputMode(termbox.InputMouse)

	l.getText()
	l.keyEvent()
}
