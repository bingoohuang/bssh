package ssh

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestClassifyCheckErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{nil, "ok"},
		{errors.New("dial tcp 127.0.0.1:22: connect: connection refused"), "refused"},
		{errors.New("dial tcp: lookup nope: no such host"), "dns"},
		{errors.New("ssh: handshake failed: ssh: unable to authenticate, attempted methods [none], no supported methods remain"), "auth failed"},
		{errors.New("ssh: handshake failed: read tcp 10.0.0.1:22: i/o timeout"), "timeout"},
		{&net.OpError{Op: "dial", Err: &timeoutError{}}, "timeout"},
		{errors.New("ssh: handshake failed: EOF"), "error"},
	}

	for i, c := range cases {
		if got := ClassifyCheckErr(c.err); got != c.want {
			t.Errorf("case %d: got %q, want %q", i, got, c.want)
		}
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestCheckOptionsNormalize(t *testing.T) {
	opt := CheckOptions{}.normalize()
	if opt.Timeout != DefaultCheckTimeout || opt.Parallel != DefaultCheckParallel || opt.Mode != CheckModeSSH {
		t.Errorf("unexpected defaults: %+v", opt)
	}

	opt = CheckOptions{Timeout: time.Second, Parallel: 4, Mode: CheckModeTCP}.normalize()
	if opt.Timeout != time.Second || opt.Parallel != 4 || opt.Mode != CheckModeTCP {
		t.Errorf("unexpected: %+v", opt)
	}
}

func TestNonEmptyServers(t *testing.T) {
	got := nonEmptyServers([]string{"", "  ", "a", "b"})
	if fmt.Sprint(got) != "[a b]" {
		t.Errorf("unexpected: %v", got)
	}
}
