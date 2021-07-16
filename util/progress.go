package util

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nanovms/ops/log"
	"github.com/tj/go-spin"
)

// ProgressSpinner is an indefinite progress indicator using a spinner.
type ProgressSpinner struct {
	spinner *spin.Spinner
	message string
	colors  log.ConsoleColorsType
	wg      *sync.WaitGroup

	done     bool
	spinning bool

	closed        bool
	interrupt     chan os.Signal
	interrupted   bool
	interruptFunc func()
}

// Start starts the spinner
func (ps *ProgressSpinner) Start(messages ...interface{}) {
	if ps.interrupted {
		return
	}

	ps.message = fmt.Sprint(messages...)
	ps.spinner = spin.New()
	ps.done = false
	ps.spinning = true
	ps.wg = &sync.WaitGroup{}
	ps.wg.Add(1)

	if ps.interruptFunc == nil {
		ps.interrupted = false
		ps.interrupt = make(chan os.Signal, 1)
		signal.Notify(ps.interrupt, os.Interrupt, syscall.SIGTERM)

		ps.interruptFunc = func() {
			<-ps.interrupt
			ps.interrupted = true
			if ps.spinning {
				ps.done = true
				ps.wg.Wait()
			}
			if !ps.closed {
				os.Exit(1)
			}
		}
		go ps.interruptFunc()
	}

	go func() {
		for {
			if ps.done {
				if ps.interrupted {
					fmt.Printf("\r%s     \n", ps.message)
				}
				break
			}

			fmt.Printf("\r%s%s %s%s", ps.colors.Yellow(), ps.spinner.Next(), ps.colors.Reset(), ps.message)
			time.Sleep(time.Millisecond * 100)
		}
		ps.wg.Done()
		ps.spinning = false
	}()
}

// Do executes given function with given messages as label.
func (ps *ProgressSpinner) Do(workFunc func() error, messages ...interface{}) error {
	ps.Start(messages...)
	if err := workFunc(); err != nil {
		ps.Fail()
		return err
	}
	ps.Done()
	return nil
}

// Done stops the spinner with success mark.
func (ps *ProgressSpinner) Done() {
	if !ps.spinning {
		return
	}
	ps.done = true
	ps.wg.Wait()
	fmt.Printf("\r%s     \n", ps.message)
}

// Fail stops the spinner with error mark.
func (ps *ProgressSpinner) Fail() {
	if !ps.spinning {
		return
	}
	ps.done = true
	ps.wg.Wait()
	fmt.Printf("\r%s     \n", ps.message)
}

// Close closes spinner and do some cleanups.
func (ps *ProgressSpinner) Close() {
	if ps.interrupt == nil {
		return
	}
	ps.closed = true
	ps.interrupt <- syscall.Signal(0)
}
