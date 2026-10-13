// Package link talks to the sign over a serial port: 9600 8N1, one frame per page, ACK/NAK wait.
package link

import (
	"errors"
	"fmt"
	"time"

	"polycomp/winapp/internal/gtxc"
)

// Logger receives one human-readable line per event (TX/RX hex, timeouts).
type Logger func(line string)

type Link struct {
	port *port
	log  Logger
}

// Ports lists serial ports (COM-ordered on Windows).
func Ports() ([]string, error) { return listPorts() }

func Open(name string, log Logger) (*Link, error) {
	p, err := openPort(name)
	if err != nil {
		return nil, err
	}
	if log == nil {
		log = func(string) {}
	}
	return &Link{port: p, log: log}, nil
}

func (l *Link) Close() error { return l.port.Close() }

var ErrNoReply = errors.New("no reply from sign")
var ErrNAK = errors.New("sign rejected the page (NAK)")

// Send writes one frame and, if expectAck, waits up to timeout for ACK/NAK.
func (l *Link) Send(frame []byte, expectAck bool, timeout time.Duration) error {
	l.log(fmt.Sprintf("TX %d bytes: %s", len(frame), gtxc.Hex(frame)))
	if _, err := l.port.Write(frame); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if !expectAck {
		return nil
	}
	_ = l.port.setReadTimeout(100 * time.Millisecond)
	deadline := time.Now().Add(timeout)
	var got []byte
	buf := make([]byte, 16)
	for time.Now().Before(deadline) {
		n, err := l.port.Read(buf)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		got = append(got, buf[:n]...)
		if n > 0 {
			break
		}
	}
	if len(got) == 0 {
		l.log("RX nothing (timeout)")
		return ErrNoReply
	}
	l.log(fmt.Sprintf("RX %d bytes: %s", len(got), gtxc.Hex(got)))
	switch got[len(got)-1] {
	case gtxc.ACK:
		return nil
	case gtxc.NAK:
		return ErrNAK
	}
	return fmt.Errorf("unexpected reply %s", gtxc.Hex(got))
}

// SendPages sends a page sequence with the more-pages bit on all but the last. The first page
// gets 3 s for its ACK, later ones 0.5 s, per the protocol notes.
func (l *Link) SendPages(frames [][]byte) error {
	for i, f := range frames {
		timeout := 500 * time.Millisecond
		if i == 0 {
			timeout = 3 * time.Second
		}
		if err := l.Send(f, true, timeout); err != nil {
			return fmt.Errorf("page %d: %w", i+1, err)
		}
	}
	return nil
}
