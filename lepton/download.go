package lepton

import (
	"time"

	pb "github.com/cheggaaa/pb"
)

const refreshRate = time.Millisecond * 100

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
	b.SetRefreshRate(refreshRate)
	b.ShowTimeLeft = true
	b.ShowSpeed = true
	b.SetUnits(pb.U_BYTES)

	return &WriteCounter{
		bar: b,
	}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	wc.n += len(p)
	wc.bar.Set(wc.n)
	return wc.n, nil
}

// Start progress bar
func (wc *WriteCounter) Start() {
	wc.bar.Start()
}

// Finish progress bar
func (wc *WriteCounter) Finish() {
	wc.bar.Finish()
}
