package main

import "github.com/zzhaolei/go-chat/internal/chat"

func main() {
	c := chat.NewChat("localhost:8080")
	if err := c.Run(); err != nil {
		panic(err)
	}
}
