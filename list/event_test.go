// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package list_test

import (
	"testing"

	"github.com/bingoohuang/bssh/list"

	"github.com/stretchr/testify/assert"
)

func TestInsertRune(t *testing.T) {
	type TestData struct {
		desc      string
		l         list.Info
		inputRune rune
		expect    string
	}

	tds := []TestData{
		{desc: "Input rune is a alphabet", l: list.Info{Keyword: "a"}, inputRune: 'b', expect: "ab"},
		{desc: "Input rune is a multibyte character", l: list.Info{Keyword: "a"}, inputRune: 'あ', expect: "aあ"},
	}

	for _, v := range tds {
		v.l.InsertRune(v.inputRune)
		assert.Equal(t, v.expect, v.l.Keyword, v.desc)
	}
}

func TestDeleteRune(t *testing.T) {
	type TestData struct {
		desc   string
		l      list.Info
		expect string
	}

	tds := []TestData{
		{desc: "Delete alphabet rune", l: list.Info{Keyword: "abc"}, expect: "ab"},
		{desc: "Delete multibyte rune", l: list.Info{Keyword: "あいう"}, expect: "あい"},
		{desc: "Expect is empty", l: list.Info{Keyword: "a"}, expect: ""},

		// nolint:godox
		// FIXME raise panic {desc: "Delete empty", l: Info{Keyword: ""}, expect: ""},
	}

	for _, v := range tds {
		v.l.DeleteRune()
		assert.Equal(t, v.expect, v.l.Keyword, v.desc)
	}
}
