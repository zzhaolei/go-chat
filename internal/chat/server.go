package chat

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
)

func Serve(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)

	c := &chat{
		l:        listener,
		wg:       sync.WaitGroup{},
		conns:    make(map[string]*client),
		shutdown: make(chan struct{}, 1),
	}
	go c.run()

	sig := <-ch
	fmt.Printf("Received signal: %s. chat closing...\n", sig)
	if err := c.close(); err != nil {
		return err
	}
	return nil
}
