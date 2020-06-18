// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package conf_test

import (
	"testing"

	"github.com/bingoohuang/bssh/conf"

	"github.com/stretchr/testify/assert"
)

func TestCheckFormatServerConf(t *testing.T) {
	type TestData struct {
		desc   string
		c      conf.Config
		expect bool
	}

	tds := []TestData{
		{
			desc: "Address, user and password",
			c: conf.Config{
				Server: map[string]conf.ServerConfig{
					"a": {Addr: "192.168.100.101", User: "test", Pass: "Password"},
				},
			},
			expect: true,
		},
		{
			desc: "Empty address",
			c: conf.Config{
				Server: map[string]conf.ServerConfig{
					"b": {Addr: "", User: "test", Pass: "Password"},
				},
			},
			expect: false,
		},
		{
			desc: "Empty user",
			c: conf.Config{
				Server: map[string]conf.ServerConfig{
					"c": {Addr: "192.168.100.101", User: "", Pass: "Password"},
				},
			},
			expect: false,
		},
		{
			desc: "Empty password",
			c: conf.Config{
				Server: map[string]conf.ServerConfig{
					"d": {Addr: "192.168.100.101", User: "test", Pass: ""},
				},
			},
			expect: false,
		},
		{
			desc: "1 server config is illegal",
			c: conf.Config{
				Server: map[string]conf.ServerConfig{
					"a": {Addr: "192.168.100.101", User: "test", Pass: "Password"},
					"b": {Addr: "", User: "test", Pass: "Password"},
					"e": {Addr: "192.168.100.101", User: "test", Pass: "Password"},
				},
			},
			expect: false,
		},
	}

	for _, v := range tds {
		got := conf.CheckFormatServerConf(v.c)
		assert.Equal(t, v.expect, got, v.desc)
	}
}

func TestCheckFormatServerConfAuth(t *testing.T) {
	type TestData struct {
		desc   string
		c      conf.ServerConfig
		expect bool
	}

	tds := []TestData{
		{desc: "Password", c: conf.ServerConfig{Pass: "Password"}, expect: true},
		{desc: "Secret key file", c: conf.ServerConfig{Key: "/tmp/key.pem"}, expect: true},
		{desc: "Cert file", c: conf.ServerConfig{Cert: "/tmp/key.crt"}, expect: true},
		{desc: "Agent auth", c: conf.ServerConfig{AgentAuth: true}, expect: true},
		// {desc: "File exists", c: ServerConfig{PKCS11Provider: "/path/to/providor"}, expect: true},
		{desc: "Key files", c: conf.ServerConfig{Keys: []string{"/tmp/key.pem", "/tmp/key2.pem"}}, expect: true},
		{desc: "Passwords", c: conf.ServerConfig{Passes: []string{"Pass1", "Pass2"}}, expect: true},
	}

	for _, v := range tds {
		got := conf.CheckFormatServerConfAuth(v.c)
		assert.Equal(t, v.expect, got, v.desc)
	}
}

func TestServerConfigReduct(t *testing.T) {
	type TestData struct {
		desc                   string
		perConfig, childConfig conf.ServerConfig
		expect                 conf.ServerConfig
	}

	tds := []TestData{
		{
			desc:        "Set perConfig addr to child config",
			perConfig:   conf.ServerConfig{Addr: "192.168.100.101", User: "pertest"},
			childConfig: conf.ServerConfig{User: "test"},
			expect:      conf.ServerConfig{Addr: "192.168.100.101", User: "test"},
		},
		{
			desc:        "Child config is empty",
			perConfig:   conf.ServerConfig{Addr: "192.168.100.101"},
			childConfig: conf.ServerConfig{},
			expect:      conf.ServerConfig{Addr: "192.168.100.101"},
		},
		{
			desc:        "Per config is empty",
			perConfig:   conf.ServerConfig{},
			childConfig: conf.ServerConfig{User: "test"},
			expect:      conf.ServerConfig{User: "test"},
		},
		{
			desc:        "Both empty",
			perConfig:   conf.ServerConfig{},
			childConfig: conf.ServerConfig{},
			expect:      conf.ServerConfig{},
		},
	}

	for _, v := range tds {
		got := conf.ServerConfigDeduct(v.perConfig, v.childConfig)
		assert.Equal(t, v.expect, got, v.desc)
	}
}

func TestGetNameList(t *testing.T) {
	type TestData struct {
		desc     string
		listConf conf.Config
		expect   []string
	}

	tds := []TestData{
		{
			desc: "",
			listConf: conf.Config{
				Server: map[string]conf.ServerConfig{
					"a": {},
					"b": {},
				},
			},
			expect: []string{"a", "b"},
		},
		{
			desc: "",
			listConf: conf.Config{
				Server: map[string]conf.ServerConfig{},
			},
			expect: nil,
		},
	}

	for _, v := range tds {
		got := v.listConf.GetNameList()
		assert.Equal(t, v.expect, got, v.desc)
	}
}
