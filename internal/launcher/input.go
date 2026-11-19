package launcher

import (
	"bytes"
	"io"
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

	if bytes.Equal(chunk, focusIn) || bytes.Equal(chunk, focusOut) {
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

func (c *counter) take() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys, pasted := c.keys, c.pasted
	c.keys, c.pasted = 0, false
	return keys, pasted
}

type countingReader struct {
	r io.Reader
	c *counter
}

func (r countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.c.observe(p[:n])
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
