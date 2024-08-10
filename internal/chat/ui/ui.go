package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/zzhaolei/go-chat/internal/chat/ctrl"
	"github.com/zzhaolei/go-chat/internal/chat/message"
	"github.com/zzhaolei/go-chat/internal/chat/tip"
)

type model struct {
	conn net.Conn

	viewport viewport.Model
	textarea textarea.Model

	serverStyle   lipgloss.Style
	senderStyle   lipgloss.Style
	receiverStyle lipgloss.Style
	messages      []string

	shutdown *chan struct{}

	err error
}

func Start(conn net.Conn) error {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	shutdown := make(chan struct{})
	m := initialModel(width, height)
	m.conn = conn
	m.shutdown = &shutdown

	ui := tea.NewProgram(m)
	go listenSrvBroadcast(ui, m.conn, shutdown)
	_, err = ui.Run()
	return err
}

func listenSrvBroadcast(ui *tea.Program, conn net.Conn, ch <-chan struct{}) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ch:
			return
		default:
		}
		n, err := conn.Read(buf)
		if err != nil {
			ui.Send(message.ServerErrMsg{Err: err})
			return
		}
		// TODO - 解析用户的名称
		var msg message.ClientMsg
		if err = json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}
		ui.Send(msg)
	}
}

func initialModel(width, height int) model {
	var (
		charLimit = 150
		minWidth  = min(charLimit, width)
		taHeight  = 3
	)

	ta := textarea.New()
	ta.Placeholder = tip.Login
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = charLimit

	ta.SetWidth(minWidth)
	ta.SetHeight(taHeight)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(minWidth, height-taHeight-20)
	vp.SetContent(tip.Login)

	return model{
		viewport:      vp,
		textarea:      ta,
		serverStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		senderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		receiverStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		messages:      []string{},
		err:           nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case message.ServerErrMsg:
		m.close()
		if errors.Is(msg.Err, io.EOF) {
			showMsg(&m, m.serverStyle.Render("Server: "+"closed."))
		} else {
			showMsg(&m, m.serverStyle.Render("Err: "+msg.Err.Error()))
		}
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.close()
			showMsg(&m, m.serverStyle.Render("Ctrl+C"))
			return m, tea.Quit
		case tea.KeyEnter:
			rawMsg := m.textarea.Value()
			if rawMsg == "" {
				return m, tea.Batch(tiCmd, vpCmd)
			}
			msg, err := parseMsg(rawMsg)
			_ = inputMsg(&m, msg, err)
		}
	case message.ClientMsg:
		switch msg.Ctrl {
		case ctrl.LoggedIn:
			showMsg(&m, tip.Welcome)

			status.name = msg.Name
			status.loggedIn = true
		case ctrl.LoggedOut:
			showMsg(&m, tip.Login)

			status.name = ""
			status.loggedIn = false
		case ctrl.Anwser:
			showMsg(&m, m.serverStyle.Render("Server: ")+msg.Msg)
		case ctrl.Broadcast:
			showMsg(&m, m.receiverStyle.Render(msg.Name+": ")+msg.Msg)
		default:
		}
	case error:
		m.err = msg
		return m, nil
	}
	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n",
		m.viewport.View(),
		m.textarea.View(),
	)
}

func (m model) close() {
	close(*m.shutdown)
}

func showMsg(m *model, msg string) {
	m.messages = append(m.messages, msg)
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
	m.viewport.GotoBottom()
}

func inputMsg(m *model, msg *message.ServerMsg, errMsg error) error {
	if errMsg != nil {
		showMsg(m, m.serverStyle.Render("Server: ")+errMsg.Error())
		m.textarea.Reset() // 设置输入状态
		return nil
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = m.conn.Write(b)
	if err != nil {
		return err
	}
	if msg.Ctrl == ctrl.Msg {
		showMsg(m, m.senderStyle.Render("You: ")+msg.Msg)
	}
	m.textarea.Reset() // 设置输入状态
	return nil
}

func parseMsg(rawMsg string) (*message.ServerMsg, error) {
	var msg message.ServerMsg
	msg.Name = status.name

	if !strings.HasPrefix(rawMsg, "/") {
		if !status.loggedIn {
			return nil, ErrCliNeedLogin
		}
		msg.Ctrl = ctrl.Msg
		msg.Msg = rawMsg
		return &msg, nil
	}

	switch {
	case strings.HasPrefix(rawMsg, "/login"):
		if status.loggedIn {
			return nil, ErrCliAlreadyLoggedIn
		}
		s := strings.Split(rawMsg, " ")
		if len(s) != 3 {
			return nil, ErrCliLogin
		}
		msg.Ctrl = ctrl.Login
		msg.Name = s[1]
		msg.Pass = s[2]
	case strings.HasPrefix(rawMsg, "/logout"):
		if !status.loggedIn {
			return nil, ErrCliAlreadyLoggedOut
		}
		msg.Ctrl = ctrl.Logout
	case strings.HasPrefix(rawMsg, "/?"):
		msg.Ctrl = ctrl.Help
	}
	return &msg, nil
}
