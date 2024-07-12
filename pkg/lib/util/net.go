package util

import (
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// WaitForTCP polls indefinitely until it can connect to the given TCP address.
func WaitForTCP(addr string) {
	logrus.Infof("Waiting for TCP address %q to become connectable...", addr)

	for {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			break
		}

		time.Sleep(10 * time.Millisecond)
	}
}
