package sshlib

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/bingoohuang/bssh/internal/tmpjson"
	"github.com/bingoohuang/ngg/ss"
)

func CreateTargetInfo(uri string, useBrg bool) (targetInfo []string, newUri string) {
	host, _, _ := net.SplitHostPort(uri)
	if ss.AnyOf(host, "127.0.0.1", "localhost") {
		return nil, uri
	}

	proxy := Getenv("PROXY", "P")
	if proxy != "" {
		proxy = " proxy=" + proxy
	}

	if useBrg && len(brg) > 0 {
		for _, p := range brg[1:] {
			targetInfo = append(targetInfo, fmt.Sprintf("TARGET %s%s;", p, proxy))
		}
		targetInfo = append(targetInfo, fmt.Sprintf("TARGET %s%s;", uri, proxy))
		return targetInfo, brg[0]
	}

	if target, ok := brgTargets[uri]; ok {
		targetInfo = append(targetInfo, fmt.Sprintf("TARGET %s%s;", uri, proxy))
		uri = target.Addr
		if strings.HasPrefix(uri, ":") {
			uri = "127.0.0.1" + uri
		}
	}

	return targetInfo, uri
}

const brgJsonFile = "brg.json"

type brgState struct {
	ProxyName   string            `json:"proxyName"`
	VisitorName string            `json:"visitorName"`
	BsshTargets map[string]Target `json:"bsshTargets"`
}

type Target struct {
	Addr string `json:"addr"`
}

var envMap = func() map[string]string {
	environ := os.Environ()
	env := make(map[string]string)
	for _, e := range environ {
		pair := strings.Split(e, "=")
		env[strings.ToUpper(pair[0])] = pair[1]
	}
	return env
}()

func Getenv(keys ...string) string {
	for _, key := range keys {
		if v := envMap[strings.ToUpper(key)]; v != "" {
			return v
		}
	}

	return ""
}

var brg, brgTargets = func() (proxies []string, targets map[string]Target) {
	brgEnv := Getenv("BRG", "B")
	if brgEnv == "" || brgEnv == "0" {
		return nil, nil
	}

	var state brgState
	_, _ = tmpjson.Read(brgJsonFile, &state)
	targets = state.BsshTargets

	parts := strings.Split(brgEnv, ",")
	for _, part := range parts {
		if len(part) == 1 {
			p, err := strconv.Atoi(part)
			if err == nil && (p >= 1 && p <= 9) {
				proxies = append(proxies, fmt.Sprintf("127.0.0.1:%d", 1000+p-1))
				continue
			}
		}

		host, port, err := net.SplitHostPort(part)
		if err != nil {
			log.Panicf("invalid %s, should [host]:port", part)
		}

		if host == "" {
			host = "127.0.0.1"
		}
		if l := port[0]; 'a' <= l && l <= 'z' || 'A' <= l && l <= 'Z' {
			portNum := parseHashedPort(port)
			port = fmt.Sprintf("%d", portNum)
		}

		proxies = append(proxies, fmt.Sprintf("%s:%s", host, port))
	}

	return
}()

func parseHashedPort(port string) uint16 {
	h := sha1.New()
	h.Write([]byte(port))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint16(sum[:2])
}

func FindPort(portStart int) int {
	for p := portStart; p < 65535; p++ {
		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			c.Close()
			return p
		}
	}

	return -1
}
