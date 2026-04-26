package misc

import "strings"

type ChanWriter struct {
	ch chan string
}

func NewChanWriter(ch chan string) *ChanWriter {
	return &ChanWriter{ch: ch}
}

func (w *ChanWriter) Write(p []byte) (n int, err error) {
	line := strings.TrimRight(string(p), "\n")
	select {
	case w.ch <- line:
	default:
		// Drop if the channel is full
	}
	return len(p), nil
}
