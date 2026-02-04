package ui

import (
	"fmt"
	"io"
	"strings"

	"tui-ssh-docker/internal/app"
	"tui-ssh-docker/internal/docker"
	"tui-ssh-docker/internal/ssh"
	"tui-ssh-docker/internal/state"
	"tui-ssh-docker/internal/vm"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gossh "golang.org/x/crypto/ssh"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#13E1B9")).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
)

type Model struct {
	app        *app.App
	vms        []vm.VMConfig
	containers []docker.ContainerInfo

	selectedVM        int
	selectedContainer int

	loading bool
	err     error
}

func InitialModel(a *app.App) Model {
	return Model{
		app: a,
	}
}

type vmsLoadedMsg []vm.VMConfig
type containersLoadedMsg []docker.ContainerInfo
type errorMsg error

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		vms, err := vm.LoadVMConfig("config/vms.json")
		if err != nil {
			return errorMsg(err)
		}
		m.app.VMs = vms // Update app state too
		return vmsLoadedMsg(vms)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.app.StateMachine.Current == state.StateVMSelector && m.selectedVM > 0 {
				m.selectedVM--
			} else if m.app.StateMachine.Current == state.StateContainerList && m.selectedContainer > 0 {
				m.selectedContainer--
			}

		case "down", "j":
			if m.app.StateMachine.Current == state.StateVMSelector && m.selectedVM < len(m.vms)-1 {
				m.selectedVM++
			} else if m.app.StateMachine.Current == state.StateContainerList && m.selectedContainer < len(m.containers)-1 {
				m.selectedContainer++
			}

		case "enter":
			if m.app.StateMachine.Current == state.StateVMSelector {
				m.loading = true
				m.app.StateMachine.Transition(state.StateConnectingVM)
				return m, func() tea.Msg {
					containers, err := m.app.LoadContainers(m.selectedVM)
					if err != nil {
						return errorMsg(err)
					}
					return containersLoadedMsg(containers)
				}
			} else if m.app.StateMachine.Current == state.StateContainerList {
				// Exec into container
				vmConfig := m.vms[m.selectedVM]
				container := m.containers[m.selectedContainer]

				return m, tea.Exec(createShellCmd(m.app.SSHManager, vmConfig, container.ID), func(err error) tea.Msg {
					if err != nil {
						return errorMsg(err)
					}
					return nil
				})
			}

		case "v": // VM Shell
			if m.app.StateMachine.Current == state.StateContainerList || m.app.StateMachine.Current == state.StateVMSelector {
				vmConfig := m.vms[m.selectedVM]
				return m, tea.Exec(createVMShellCmd(m.app.SSHManager, vmConfig), func(err error) tea.Msg {
					if err != nil {
						return errorMsg(err)
					}
					return nil
				})
			}

		case "esc", "backspace":
			if m.app.StateMachine.Current == state.StateContainerList {
				m.app.StateMachine.Transition(state.StateVMSelector)
				m.containers = nil
				m.selectedContainer = 0
			}
		}

	case vmsLoadedMsg:
		m.vms = msg
		m.app.StateMachine.Transition(state.StateVMSelector)
		m.loading = false

	case containersLoadedMsg:
		m.containers = msg
		m.app.StateMachine.Transition(state.StateContainerList)
		m.loading = false

	case errorMsg:
		m.err = msg
		m.app.StateMachine.SetError(msg)
		m.loading = false
	}

	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render(" TUI SSH DOCKER "))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\nPress 'q' to quit.")
		return s.String()
	}

	if m.loading {
		s.WriteString(statusStyle.Render("Loading..."))
		return s.String()
	}

	switch m.app.StateMachine.Current {
	case state.StateVMSelector:
		s.WriteString("Select a Virtual Machine:\n\n")
		for i, v := range m.vms {
			if i == m.selectedVM {
				s.WriteString(selectedItemStyle.Render(fmt.Sprintf("-> %s (%s)", v.Name, v.Host)))
			} else {
				s.WriteString(itemStyle.Render(fmt.Sprintf("   %s (%s)", v.Name, v.Host)))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n" + statusStyle.Render("[↑/↓: Navigate, Enter: Connect, q: Quit]"))

	case state.StateContainerList:
		vm := m.vms[m.selectedVM]
		s.WriteString(fmt.Sprintf("Containers on %s (%s):\n\n", vm.Name, vm.Host))
		if len(m.containers) == 0 {
			s.WriteString(itemStyle.Render("No containers found."))
		} else {
			for i, c := range m.containers {
				if i == m.selectedContainer {
					s.WriteString(selectedItemStyle.Render(fmt.Sprintf("-> %s [%s] (%s)", c.Name, c.Image, c.Status)))
				} else {
					s.WriteString(itemStyle.Render(fmt.Sprintf("   %s [%s] (%s)", c.Name, c.Image, c.Status)))
				}
				s.WriteString("\n")
			}
		}
		s.WriteString("\n" + statusStyle.Render("[↑/↓: Navigate, Enter: Exec, v: VM Shell, Esc: Back, q: Quit]"))
	}

	return s.String()
}

type SSHShellCmd struct {
	manager *ssh.SSHManager
	vm      vm.VMConfig
	command string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (c *SSHShellCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *SSHShellCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *SSHShellCmd) SetStderr(w io.Writer) { c.stderr = w }

func (c *SSHShellCmd) Run() error {
	client, err := c.manager.Connect(c.vm)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = c.stdin
	session.Stdout = c.stdout
	session.Stderr = c.stderr

	// Request PTY
	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}

	// Dynamic terminal size would be better, but let's start with a decent default
	if err := session.RequestPty("xterm-256color", 40, 80, modes); err != nil {
		return err
	}

	if c.command != "" {
		if err := session.Start(c.command); err != nil {
			return err
		}
	} else {
		if err := session.Shell(); err != nil {
			return err
		}
	}

	return session.Wait()
}

func createShellCmd(manager *ssh.SSHManager, v vm.VMConfig, containerID string) *SSHShellCmd {
	return &SSHShellCmd{
		manager: manager,
		vm:      v,
		command: fmt.Sprintf("docker exec -it %s bash || docker exec -it %s sh", containerID, containerID),
	}
}

func createVMShellCmd(manager *ssh.SSHManager, v vm.VMConfig) *SSHShellCmd {
	return &SSHShellCmd{
		manager: manager,
		vm:      v,
		command: "", // interactive shell
	}
}
