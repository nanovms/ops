package lepton

import (
	"fmt"

	pb "github.com/schollz/progressbar/v2"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer
// interface and we can pass this into io.TeeReader() which will report progress on each
// write cycle.
type WriteCounter struct {
	n   int // bytes read so far
	bar *pb.ProgressBar
}

// NewWriteCounter creates new write counter
func NewWriteCounter(total int) *WriteCounter {
	b := pb.New(total)
	return &WriteCounter{
		bar: b,
	}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	wc.bar.Add(len(p))
	return wc.n, nil
}

// Start progress bar
func (wc *WriteCounter) Start() {
	wc.bar.RenderBlank()
}

// Finish progress bar
func (wc *WriteCounter) Finish() {
	wc.bar.Finish()
	fmt.Printf("\n")
}
