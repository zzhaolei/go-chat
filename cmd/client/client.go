package main

import (
	"github.com/zzhaolei/go-chat/internal/chat"
)

func main() {
	if err := chat.Dial("localhost:8080"); err != nil {
		panic(err)
	}
}
