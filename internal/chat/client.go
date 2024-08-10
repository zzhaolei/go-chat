package chat

import (
	"errors"
	"fmt"
	"net"

	"github.com/zzhaolei/go-chat/internal/chat/ui"
)

func Dial(addr string) (err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer func() {
		if closeE := conn.Close(); closeE != nil {
			err = errors.Join(err, closeE)
		}
	}()

	err = ui.Start(conn)
	fmt.Printf("chat client closed.")
	return err
}
