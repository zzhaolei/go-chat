package ctrl

type Ctrl int

const (
	Help Ctrl = iota + 1
	Anwser
	Msg
	Login
	Logout
	LoggedIn
	LoggedOut
	Broadcast
)
