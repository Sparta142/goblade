package internal

import "io"

type channelReader struct {
	ch   <-chan []byte
	data []byte
}

// https://github.com/google/gopacket/blob/3eaba08/examples/reassemblydump/main.go#L95
func (rd *channelReader) Read(p []byte) (n int, err error) {
	ok := true
	for ok && len(rd.data) == 0 {
		rd.data, ok = <-rd.ch
	}

	if !ok || len(rd.data) == 0 {
		return 0, io.EOF
	}

	l := copy(p, rd.data)
	rd.data = rd.data[l:]
	return l, nil
}

// Creates an io.Reader from a channel of []byte.
func NewChannelReader(ch <-chan []byte) *channelReader {
	return &channelReader{ch: ch}
}
