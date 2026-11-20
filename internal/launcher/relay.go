package launcher

import (
	"io"
	"time"
)

// copies in -> term and term -> out until wait returns, counting input on the way.
// term is the harness's pty (unix) or conpty pipes (windows)
func relay(term io.ReadWriter, wait func() (int, error), in io.Reader, out io.Writer, o Options) (int, error) {
	var keys counter
	stop, ticked := make(chan struct{}), make(chan struct{})
	go ticker(&keys, tickEvery, o.OnTick, stop, ticked)

	go io.Copy(term, countingReader{in, &keys, &focusWatcher{onFocus: o.OnFocus}})

	done := make(chan struct{})
	go func() {
		io.Copy(out, term) // ends with eof or eio once the harness's side closes
		close(done)
	}()

	code, err := wait()

	// let the last output through, but don't hang if a leftover child keeps it open
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
	}
	close(stop)
	<-ticked
	return code, err
}
