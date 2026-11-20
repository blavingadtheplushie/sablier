package opencode

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// opencode server to grab events
type Server struct {
	URL      string
	password string
	cmd      *exec.Cmd
	stopOnce sync.Once
	exited   chan struct{}
}

var listening = regexp.MustCompile(`listening on (http://\S+)`)

func Start(bin string) (*Server, error) {
	path, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("%s not found in PATH", bin)
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	s := &Server{password: hex.EncodeToString(b), exited: make(chan struct{})}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	s.cmd = exec.Command(path, "serve", "--hostname", "127.0.0.1", "--port", "0")
	s.cmd.Env = append(os.Environ(), "OPENCODE_PASSWORD="+s.password)
	s.cmd.Stdout, s.cmd.Stderr = w, w
	err = s.cmd.Start()
	w.Close()
	if err != nil {
		r.Close()
		return nil, err
	}
	go func() { s.cmd.Wait(); close(s.exited) }()

	url := make(chan string, 1)
	go func() {
		defer r.Close()
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			if m := listening.FindStringSubmatch(sc.Text()); m != nil {
				url <- m[1]
				break
			}
		}
		io.Copy(io.Discard, r)
	}()

	select {
	case s.URL = <-url:
		return s, nil
	case <-s.exited:
		return nil, errors.New("opencode server exited before it was ready")
	case <-time.After(15 * time.Second):
		s.Stop()
		return nil, errors.New("opencode server didn't start in 15s")
	}
}

func (s *Server) Env() []string { return []string{"OPENCODE_PASSWORD=" + s.password} }

func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		if s.cmd.Process.Signal(os.Interrupt) != nil { // windows can't
			s.cmd.Process.Kill()
		}
		select {
		case <-s.exited:
		case <-time.After(3 * time.Second):
			s.cmd.Process.Kill()
			<-s.exited
		}
	})
}

func (s *Server) Watch(ctx context.Context, fn func(Event)) {
	for ctx.Err() == nil {
		s.stream(ctx, fn)
		select {
		case <-ctx.Done():
		case <-s.exited:
			return
		case <-time.After(time.Second):
		}
	}
}

func (s *Server) stream(ctx context.Context, fn func(Event)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.URL+"/api/event", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth("opencode", s.password)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("event stream: %s", resp.Status)
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 32<<20)
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data:")
		if !ok {
			continue
		}
		var e Event
		if json.Unmarshal([]byte(data), &e) == nil {
			fn(e)
		}
	}
	return sc.Err()
}

func RunsSession(args []string) bool {
	return len(args) == 0 || strings.HasPrefix(args[0], "-") || isDir(args[0]) || args[0] == "run"
}

func WithServer(url string, args []string) []string {
	if len(args) > 0 && args[0] == "run" {
		return append([]string{"run", "--server", url}, args[1:]...)
	}
	return append([]string{"--server", url}, args...)
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
