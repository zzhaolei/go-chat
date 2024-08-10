package ui

import "errors"

var (
	ErrCliAlreadyLoggedIn  = errors.New("already logged in")
	ErrCliAlreadyLoggedOut = errors.New("already logged out")
	ErrCliLogin            = errors.New("login error")
	ErrInternal            = errors.New("internal error")
	ErrCliNeedLogin        = errors.New("need login")
)

type ServerErrMsg struct {
	Err error
}
