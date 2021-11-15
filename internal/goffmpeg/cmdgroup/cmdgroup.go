// Package cmdgroup runs and terminate commands as a group. Similar to errgroup.
package cmdgroup

import (
	"context"
	"sync"
)

// Cmd somthing that can start and wait to finish, like exec.Cmd.
// Must respect group context.
type Cmd interface {
	Start() error
	Wait() error
}

// Group of commands that runs and terminate as a group. Similar to errgroup.
type Group struct {
	cancelFn func()
	cmds     []Cmd
}

// WithContext creates a new command group
func WithContext(parentCtx context.Context) (*Group, context.Context) {
	ctx, cancelFn := context.WithCancel(parentCtx)
	return &Group{cancelFn: cancelFn}, ctx
}

// Add cmd to group
func (g *Group) Add(cmd Cmd) {
	g.cmds = append(g.cmds, cmd)
}

// Run commands in group plus commands given as argument.
func (g *Group) Run(cmds ...Cmd) []error {
	var cancelOnce sync.Once

	g.cmds = append(g.cmds, cmds...)

	var startErrs []error
	for _, cmd := range g.cmds {
		if err := cmd.Start(); err != nil {
			startErrs = append(startErrs, err)
		}
	}

	if len(startErrs) > 0 {
		cancelOnce.Do(g.cancelFn)
	}

	var waitErrs []error
	waitErrChan := make(chan error)
	for _, cmd := range g.cmds {
		go func(cmd Cmd) {
			waitErrChan <- cmd.Wait()
		}(cmd)
	}
	// there will be len(cmds) errors
	for range cmds {
		err := <-waitErrChan
		if err != nil {
			waitErrs = append(waitErrs, err)
			cancelOnce.Do(g.cancelFn)
		}
	}

	cancelOnce.Do(g.cancelFn)

	return waitErrs
}
