package systemd

import (
	"time"
)

func timeoutJobExecution() chan bool {
	timeout := make(chan bool, 1)

	go func() {
		time.Sleep(60 * time.Second)
		timeout <- true
	}()

	return timeout
}

