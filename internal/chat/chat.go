package chat

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

type client struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	conn     net.Conn
}

type Chat struct {
	l        net.Listener
	wg       sync.WaitGroup
	conns    map[string]client
	shutdown chan struct{}
}

func NewChat(addr string) *Chat {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	return &Chat{
		l:        listener,
		wg:       sync.WaitGroup{},
		conns:    make(map[string]client),
		shutdown: make(chan struct{}, 1),
	}
}

func (c *Chat) close() error {
	close(c.shutdown)

	if err := c.l.Close(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		done <- struct{}{}
	}()
	select {
	case <-done:
		return nil
	case <-time.After(time.Second * 5):
		return errors.New("server shutdown timeout")
	}
}

func (c *Chat) Run() (err error) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)

	go c.run()

	sig := <-ch
	fmt.Printf("Received signal: %s. Chat closing...\n", sig)

	if err := c.close(); err != nil {
		panic(err)
	}
	return nil
}

func (c *Chat) run() {
	fmt.Println("Listening on:", c.l.Addr())

	for {
		conn, err := c.l.Accept()
		if err != nil {
			select {
			case <-c.shutdown:
				return
			default:
				fmt.Println("accept error:", err)
				continue
			}
		}

		c.wg.Add(1)
		go c.handle(conn)
	}
}

func (c *Chat) handle(conn net.Conn) {
	defer func() {
		c.wg.Done()
		if err := conn.Close(); err != nil {
			fmt.Printf("close conn err: %v\n", err)
		}
	}()

	fmt.Println("connection opened:", conn.RemoteAddr())

	buf := make([]byte, 1024)
	for {
		select {
		case <-c.shutdown:
			return
		default:
		}

		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("connection closed:", conn.RemoteAddr())
				return
			}
			fmt.Println("conn read error:", err)
		}
		_, err = conn.Write(buf[:n])
		if err != nil {
			fmt.Println("write error:", err)
		}
	}
}
