//go:build windows

package launcher

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

// the harness gets its own conpty (windows 10 1809+) and we sit in between to intercept focus/typing
func runTerminal(c *exec.Cmd, o Options) (int, error) {
	stdin := windows.Handle(os.Stdin.Fd())
	stdout := windows.Handle(os.Stdout.Fd())

	size, err := consoleSize(stdout)
	if err != nil {
		return runDirect(c)
	}

	var ptyIn, toPty, fromPty, ptyOut windows.Handle
	if err := windows.CreatePipe(&ptyIn, &toPty, nil, 0); err != nil {
		return 0, err
	}
	if err := windows.CreatePipe(&fromPty, &ptyOut, nil, 0); err != nil {
		closeHandles(ptyIn, toPty)
		return 0, err
	}
	var hpc windows.Handle
	if err := windows.CreatePseudoConsole(size, ptyIn, ptyOut, 0, &hpc); err != nil {
		closeHandles(ptyIn, toPty, fromPty, ptyOut)
		return 0, err
	}
	closeHandles(ptyIn, ptyOut)

	toFile := os.NewFile(uintptr(toPty), "conpty-in")
	fromFile := os.NewFile(uintptr(fromPty), "conpty-out")
	defer toFile.Close()
	defer fromFile.Close()

	proc, err := startInConsole(c, hpc)
	if err != nil {
		windows.ClosePseudoConsole(hpc)
		return 0, err
	}
	defer windows.CloseHandle(proc)

	var outMode uint32
	if windows.GetConsoleMode(stdout, &outMode) == nil {
		windows.SetConsoleMode(stdout, outMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING|windows.DISABLE_NEWLINE_AUTO_RETURN)
		defer windows.SetConsoleMode(stdout, outMode)
	}

	// raw with vt input
	// ctrl-c is a byte the harness's conpty turns into its own ctrl-c
	old, err := term.MakeRaw(int(stdin))
	if err != nil {
		windows.TerminateProcess(proc, 1)
		windows.ClosePseudoConsole(hpc)
		return 0, err
	}
	defer term.Restore(int(stdin), old)

	stopResize, resized := make(chan struct{}), make(chan struct{})
	go watchSize(stdout, hpc, size, stopResize, resized)

	var closeOnce sync.Once
	wait := func() (int, error) {
		windows.WaitForSingleObject(proc, windows.INFINITE)
		var code uint32
		err := windows.GetExitCodeProcess(proc, &code)
		closeOnce.Do(func() {
			close(stopResize)
			<-resized // no resize can race the close
			windows.ClosePseudoConsole(hpc)
		})
		return int(code), err
	}
	return relay(struct {
		io.Reader
		io.Writer
	}{fromFile, toFile}, wait, os.Stdin, os.Stdout, o)
}

func startInConsole(c *exec.Cmd, hpc windows.Handle) (windows.Handle, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return 0, err
	}
	defer attrs.Delete()
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&hpc)), unsafe.Sizeof(hpc)); err != nil {
		return 0, err
	}

	si := windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(si))

	app, err := windows.UTF16PtrFromString(c.Path)
	if err != nil {
		return 0, err
	}
	args := append([]string{c.Path}, c.Args[1:]...)
	cmdLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(args))
	if err != nil {
		return 0, err
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(app, cmdLine, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		envBlock(c.Env), nil, &si.StartupInfo, &pi)
	if err != nil {
		return 0, err
	}
	windows.CloseHandle(pi.Thread)
	return pi.Process, nil
}

// "k=v\0k=v\0\0" in utf-16
func envBlock(env []string) *uint16 {
	var b []uint16
	for _, kv := range env {
		b = append(b, utf16.Encode([]rune(kv))...)
		b = append(b, 0)
	}
	b = append(b, 0)
	if len(env) == 0 {
		b = append(b, 0)
	}
	return &b[0]
}

func consoleSize(out windows.Handle) (windows.Coord, error) {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(out, &info); err != nil {
		return windows.Coord{}, err
	}
	return windows.Coord{
		X: info.Window.Right - info.Window.Left + 1,
		Y: info.Window.Bottom - info.Window.Top + 1,
	}, nil
}

// windows has no SIGWINCH, so poll
func watchSize(out, hpc windows.Handle, last windows.Coord, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if s, err := consoleSize(out); err == nil && s != last {
				windows.ResizePseudoConsole(hpc, s)
				last = s
			}
		}
	}
}

func closeHandles(hs ...windows.Handle) {
	for _, h := range hs {
		windows.CloseHandle(h)
	}
}
