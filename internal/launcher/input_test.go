package launcher

import (
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	var c counter
	c.observe([]byte("a"))
	c.observe([]byte("\x1b[A")) // arrow key, one chunk
	c.observe(focusIn)
	c.observe(focusOut)
	if keys, pasted := c.take(); keys != 2 || pasted {
		t.Fatalf("got %d keys, pasted %v", keys, pasted)
	}
	if keys, _ := c.take(); keys != 0 {
		t.Fatal("take should reset")
	}
}

func TestCounterPaste(t *testing.T) {
	var c counter
	c.observe([]byte("\x1b[200~first half"))
	c.observe([]byte("second half\x1b[201~"))
	c.observe([]byte("x"))
	if keys, pasted := c.take(); keys != 1 || !pasted {
		t.Fatalf("got %d keys, pasted %v", keys, pasted)
	}
}

func TestTicker(t *testing.T) {
	var c counter
	ticks := make(chan Tick, 10)
	stop, done := make(chan struct{}), make(chan struct{})
	go ticker(&c, 20*time.Millisecond, func(tk Tick) { ticks <- tk }, stop, done)

	c.observe([]byte("a"))
	time.Sleep(50 * time.Millisecond) // a window with input, then an empty one
	c.observe([]byte("b"))
	close(stop)
	<-done
	close(ticks)

	var got []int
	for tk := range ticks {
		got = append(got, tk.Keys)
		if tk.Window <= 0 {
			t.Errorf("bad window %s", tk.Window)
		}
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Fatalf("got ticks %v, want [1 1] (empty windows skipped)", got)
	}
}

func TestFocusWatcher(t *testing.T) {
	var got []bool
	f := focusWatcher{onFocus: func(in bool, _ time.Time) { got = append(got, in) }}
	for _, chunk := range []string{"\x1b[I", "a", "\x1b[I", "\x1b[O", "\x1b[O", "\x1b[O\x1b[I"} {
		f.observe([]byte(chunk))
	}
	// in, (key), repeat in skipped, out, repeat out skipped, out then in in one chunk ends in
	if len(got) != 3 || !got[0] || got[1] || !got[2] {
		t.Fatalf("got %v, want [true false true]", got)
	}
}

func TestCounterIgnoresMouse(t *testing.T) {
	var c counter
	c.observe([]byte("\x1b[<64;30;12M"))                // wheel up, sgr
	c.observe([]byte("\x1b[<65;30;12M\x1b[<65;30;13M")) // fast scroll, two in one chunk
	c.observe([]byte("\x1b[<0;10;5M\x1b[<0;10;5m"))     // click press + release
	c.observe([]byte("\x1b[M`!!"))                      // wheel up, x10
	c.observe([]byte("\x1b[O\x1b[<64;1;1M"))            // focus + scroll together
	if keys, _ := c.take(); keys != 0 {
		t.Fatalf("mouse counted as %d keys", keys)
	}

	c.observe([]byte("a\x1b[<64;30;12M")) // typing mixed in still counts
	if keys, _ := c.take(); keys != 1 {
		t.Fatalf("got %d keys, want 1", keys)
	}
}
