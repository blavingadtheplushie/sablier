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
