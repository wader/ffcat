// Package execextra is go stdlib exec.Cmd with some ExtraFiles additions.
// most code based on go stdlib exec package.
package execextra

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
)

type closeOnce struct {
	*os.File

	once sync.Once
	err  error
}

func (c *closeOnce) Close() error {
	c.once.Do(c.close)
	return c.err
}

func (c *closeOnce) close() {
	c.err = c.File.Close()
}

// Command see go stdlib exec.Command
func Command(name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.Command(name, arg...)}
}

// CommandContext see go stdlib exec.CommandContext
func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.CommandContext(ctx, name, arg...)}
}

// Cmd is a go stdlib exec.Cmd with some ExtraFiles additions
type Cmd struct {
	*exec.Cmd

	// design borrowed from go src/exec/exec.go
	closeAfterStart []io.Closer
	closeAfterWait  []io.Closer
	copyErrCh       chan error
	copyFns         []func() error
}

func (c *Cmd) closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		fd.Close()
	}
}

// Start see go stdlib cmd.Start
func (c *Cmd) Start() error {
	if err := c.Cmd.Start(); err != nil {
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
		return err
	}
	c.closeDescriptors(c.closeAfterStart)

	c.copyErrCh = make(chan error, len(c.copyFns))
	for _, fn := range c.copyFns {
		go func(fn func() error) {
			c.copyErrCh <- fn()
		}(fn)
	}

	return nil
}

// Run see go stdlib cmd.Run
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Wait see go stdlib cmd.Wait
func (c *Cmd) Wait() error {
	err := c.Cmd.Wait()

	var copyErr error
	for range c.copyFns {
		if err := <-c.copyErrCh; err != nil && copyErr == nil {
			copyErr = err
		}
	}
	c.closeDescriptors(c.closeAfterWait)

	if err != nil {
		return err
	}

	return copyErr
}

// ExtraInPipe returns a pipe and fd number that will be connected to a
// readable fd in the child process.
//
// See StdinPipe for how to handle close and exit.
func (c *Cmd) ExtraInPipe() (wc io.WriteCloser, childFD uintptr, err error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, 0, err
	}
	c.ExtraFiles = append(c.ExtraFiles, pr)
	c.closeAfterStart = append(c.closeAfterStart, pr)
	wc = &closeOnce{File: pw}
	c.closeAfterWait = append(c.closeAfterWait, wc)
	return wc, uintptr(len(c.ExtraFiles)) + 2, nil
}

// ExtraIn connects a io.Reader to a readable fd in the child process.
//
// Simlar to cmd.Stdin.
func (c *Cmd) ExtraIn(r io.Reader) (childFD uintptr, err error) {
	if f, ok := r.(*os.File); ok {
		c.ExtraFiles = append(c.ExtraFiles, f)
		// child fd will be index+3, see cmd.ExtraFiles
		return uintptr(len(c.ExtraFiles)) + 2, nil
	}

	wc, fd, err := c.ExtraInPipe()
	if err != nil {
		return 0, err
	}
	c.copyFns = append(c.copyFns, func() error {
		_, err := io.Copy(wc, r)
		wc.Close()
		return err
	})

	return fd, nil
}

// ExtraOutPipe returns a pipe and a fd number that will be connected to a
// writable fd in the child process.
//
// See StdoutPipe for how to handle close and exit.
func (c *Cmd) ExtraOutPipe() (rc io.ReadCloser, childFD uintptr, err error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, 0, err
	}
	c.ExtraFiles = append(c.ExtraFiles, pw)
	c.closeAfterStart = append(c.closeAfterStart, pw)
	c.closeAfterWait = append(c.closeAfterWait, pr)
	return pr, uintptr(len(c.ExtraFiles)) + 2, nil
}

// ExtraOut connects a io.Writer to a writable fd in the child process.
//
// Simlar to cmd.Stdout.
func (c *Cmd) ExtraOut(w io.Writer) (childFD uintptr, err error) {
	if f, ok := w.(*os.File); ok {
		c.ExtraFiles = append(c.ExtraFiles, f)
		return uintptr(len(c.ExtraFiles)) + 2, nil
	}

	rc, fd, err := c.ExtraOutPipe()
	if err != nil {
		return 0, err
	}
	c.copyFns = append(c.copyFns, func() error {
		_, err := io.Copy(w, rc)
		rc.Close()
		return err
	})

	return fd, nil
}

// CloseAfterStart adds an additional closer to close after command start.
// Useful to close read/write end when piping between processes.
func (c *Cmd) CloseAfterStart(closer io.Closer) {
	c.closeAfterStart = append(c.closeAfterStart, closer)
}

// CloseAfterWait adds an additional closer to close after command wait
// Useful to close read/write end when piping between processes.
func (c *Cmd) CloseAfterWait(closer io.Closer) {
	c.closeAfterWait = append(c.closeAfterWait, closer)
}
