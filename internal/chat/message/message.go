package message

import "github.com/zzhaolei/go-chat/internal/chat/ctrl"

type ServerErrMsg struct {
	Err error
}

type ServerMsg struct {
	Name string    `json:"name"` // unique
	Ctrl ctrl.Ctrl `json:"ctrl"`
	Pass string    `json:"pass,omitempty"`
	Msg  string    `json:"msg,omitempty"`
}

type ClientMsg struct {
	Name string    `json:"name"` // unique
	Ctrl ctrl.Ctrl `json:"ctrl"`
	Msg  string    `json:"msg,omitempty"`
}
