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

func CreateTargetInfo(uri, confBrg string) (targetInfo []string, newUri string) {
	host, _, _ := net.SplitHostPort(uri)
	if ss.AnyOf(host, "127.0.0.1", "localhost") {
		return nil, uri
	}

	proxy := Getenv("BRG_PROXY") // 使用 BRG_PROXY 指示 brg 优先使用改名字的代理
	if proxy != "" {
		proxy = " proxy=" + proxy
	}

	var localBrg = brg
	if confBrg == "0" {
		localBrg = nil
	} else if confBrg != "" {
		localBrg = createBrgProxies(confBrg)
	}

	if len(localBrg) > 0 {
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
	// 在显示指定 PROXY 时，不使用 BRG
	// 例如：export PROXY=socks5://127.0.0.1:6000
	if env := Getenv("PROXY"); env != "" {
		return nil, nil
	}
	brgEnv := Getenv("BRG")
	if brgEnv == "" || brgEnv == "0" {
		return nil, nil
	}

	var state brgState
	_, _ = tmpjson.Read(brgJsonFile, &state)
	targets = state.BsshTargets
	proxies = createBrgProxies(brgEnv)
	return
}()

func createBrgProxies(brgEnv string) (proxies []string) {
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

	return proxies
}

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
