package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/zzhaolei/go-chat/internal/chat/ctrl"
	"github.com/zzhaolei/go-chat/internal/chat/message"
	"github.com/zzhaolei/go-chat/internal/chat/tip"
)

type client struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	isLogin  bool
	conn     net.Conn
}

type chat struct {
	l        net.Listener
	wg       sync.WaitGroup
	conns    map[string]*client
	shutdown chan struct{}
}

func (c *chat) close() error {
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
		return errors.New("chat shutdown timeout")
	}
}

func (c *chat) run() {
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
		go c.handleConn(conn)
	}
}

func (c *chat) handleConn(conn net.Conn) {
	defer func() {
		c.wg.Done()
		if err := conn.Close(); err != nil {
			fmt.Printf("close conn err: %v\n", err)
		}
		fmt.Println("close conn")
	}()

	fmt.Println("open conn:", conn.RemoteAddr())

	buf := make([]byte, 1024)
	for {
		select {
		case <-c.shutdown:
			return
		default:
		}

		sm, err := c.readMsg(conn, buf)
		if err != nil {
			var ne *net.OpError
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("read msg error:", err)
			continue
		}

		err = c.handleMsg(conn, sm)
		if err != nil {
			fmt.Println("handleMsg error:", err)
			return
		}
	}
}

func (c *chat) readMsg(conn net.Conn, buf []byte) (*message.ServerMsg, error) {
	_ = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	fmt.Printf("from: %s, read: %s\n", conn.RemoteAddr().String(), string(buf[:n]))

	var sm message.ServerMsg
	if err := json.Unmarshal(buf[:n], &sm); err != nil {
		return nil, err
	}
	return &sm, nil
}

func (c *chat) SendMsg(conn net.Conn, rm message.ClientMsg) error {
	b, err := json.Marshal(rm)
	if err != nil {
		return err
	}
	if _, err = conn.Write(b); err != nil {
		return err
	}
	return nil
}

func (c *chat) handleMsg(conn net.Conn, sm *message.ServerMsg) error {
	switch sm.Ctrl {
	case ctrl.Help:
		if err := c.SendMsg(conn, message.ClientMsg{
			Ctrl: ctrl.Anwser,
			Msg:  tip.Help,
		}); err != nil {
			return err
		}
	case ctrl.Login:
		id := sm.Name
		user, ok := c.conns[id]
		if !ok {
			user = &client{
				Name:     id,
				Password: sm.Pass,
				isLogin:  true,
				conn:     conn,
			}
		} else {
			// TODO - 不允许重复登录，应该返回一个异常
			if user.isLogin {
				return nil
			}
		}
		c.conns[id] = user

		c.broadcastMsg(sm.Name, "Logged in")
		if err := c.SendMsg(conn, message.ClientMsg{
			Name: user.Name,
			Ctrl: ctrl.LoggedIn,
		}); err != nil {
			return err
		}
	case ctrl.Logout:
		user, ok := c.conns[sm.Name]
		// TODO - 未登录
		if !ok {
			return nil
		}

		c.broadcastMsg(sm.Name, "Logged out")
		if err := c.SendMsg(conn, message.ClientMsg{
			Name: user.Name,
			Ctrl: ctrl.LoggedOut,
		}); err != nil {
			fmt.Println("send login error:", err)
		}
		user.isLogin = false
		fmt.Printf("name=%s logout\n", user.Name)
	case ctrl.Msg:
		c.broadcastMsg(sm.Name, sm.Msg)
	default:
		fmt.Println("unhandled default case")
	}
	return nil
}

func (c *chat) broadcastMsg(sendName string, sendMsg string) {
	for _, v := range c.conns {
		if v.Name == sendName {
			continue
		}

		cm := message.ClientMsg{
			Name: sendName,
			Ctrl: ctrl.Broadcast,
			Msg:  sendMsg,
		}
		if err := c.SendMsg(v.conn, cm); err != nil {
			fmt.Printf("write to name=%s error: %v\n", v.Name, err)
		}
	}
}
