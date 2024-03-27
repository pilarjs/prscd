//go:build darwin
// +build darwin

package websocket

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// reuse port on darwin
var lc = net.ListenConfig{
	Control: func(_, _ string, c syscall.RawConn) error {
		var opErr error
		if err := c.Control(func(fd uintptr) {
			// reuse port, this can listen on the same port in multiple processes, make full use of CPU resources; and can also achieve hot update
			opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		}); err != nil {
			return err
		}
		return opErr
	},
}
