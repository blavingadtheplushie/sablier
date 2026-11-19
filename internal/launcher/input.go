package launcher

import (
	"bytes"
	"io"
	"regexp"
	"sync"
	"time"
)

type Tick struct {
	At     time.Time // end of the window
	Window time.Duration
	Keys   int
	Pasted bool
}

var (
	focusIn    = []byte("\x1b[I")
	focusOut   = []byte("\x1b[O")
	pasteStart = []byte("\x1b[200~")
	pasteEnd   = []byte("\x1b[201~")
)

// counts input chunks, never keeps their content
type counter struct {
	mu      sync.Mutex
	keys    int
	pasted  bool
	inPaste bool
}

func (c *counter) observe(chunk []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.inPaste && len(withoutNoise(chunk)) == 0 {
		return
	}
	if bytes.Contains(chunk, pasteStart) {
		c.inPaste = true
	}
	if c.inPaste {
		c.pasted = true
	} else {
		c.keys++
	}
	if bytes.Contains(chunk, pasteEnd) {
		c.inPaste = false
	}
}

// mouse reports: sgr (esc [ < b ; x ; y M/m) and x10 (esc [ M + 3 bytes)
var mouseSeq = regexp.MustCompile(`\x1b\[<\d+;\d+;\d+[Mm]|\x1b\[M[\x00-\xff]{3}`)

// drops what the terminal sends on its own: focus changes and mouse/scroll reports
func withoutNoise(chunk []byte) []byte {
	chunk = mouseSeq.ReplaceAll(chunk, nil)
	chunk = bytes.ReplaceAll(chunk, focusIn, nil)
	return bytes.ReplaceAll(chunk, focusOut, nil)
}

func (c *counter) take() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys, pasted := c.keys, c.pasted
	c.keys, c.pasted = 0, false
	return keys, pasted
}

// terminal focus, from the sequences the terminal sends once the harness enables focus reporting
type focusWatcher struct {
	known, in bool
	onFocus   func(in bool, at time.Time)
}

func (f *focusWatcher) observe(chunk []byte) {
	i, o := bytes.LastIndex(chunk, focusIn), bytes.LastIndex(chunk, focusOut)
	if i < 0 && o < 0 {
		return
	}
	in := i > o
	if f.known && f.in == in {
		return // only changes
	}
	f.known, f.in = true, in
	f.onFocus(in, time.Now())
}

type countingReader struct {
	r io.Reader
	c *counter
	f *focusWatcher
}

func (r countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.c.observe(p[:n])
		r.f.observe(p[:n])
	}
	return n, err
}

// sends a tick every interval while there was input, and a last one on stop
func ticker(c *counter, interval time.Duration, onTick func(Tick), stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	last := time.Now()
	flush := func(now time.Time) {
		if keys, pasted := c.take(); keys > 0 || pasted {
			onTick(Tick{At: now, Window: now.Sub(last), Keys: keys, Pasted: pasted})
		}
		last = now
	}

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case now := <-t.C:
			flush(now)
		case <-stop:
			flush(time.Now())
			return
		}
	}
}
