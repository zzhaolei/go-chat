package main

import (
	"github.com/zzhaolei/go-chat/internal/chat"
)

func main() {
	if err := chat.Serve("localhost:8080"); err != nil {
		panic(err)
	}
}
