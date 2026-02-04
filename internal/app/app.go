package app

import (
	"fmt"
	"tui-ssh-docker/internal/docker"
	"tui-ssh-docker/internal/ssh"
	"tui-ssh-docker/internal/state"
	"tui-ssh-docker/internal/vm"
)

type App struct {
	SSHManager       *ssh.SSHManager
	DockerController *docker.DockerController
	StateMachine     *state.StateMachine
	VMs              []vm.VMConfig
}

func NewApp() (*App, error) {
	sshManager := ssh.NewSSHManager()
	return &App{
		SSHManager:       sshManager,
		DockerController: docker.NewDockerController(sshManager),
		StateMachine:     state.NewStateMachine(),
	}, nil
}

func (a *App) LoadConfig(path string) error {
	vms, err := vm.LoadVMConfig(path)
	if err != nil {
		a.StateMachine.SetError(err)
		return err
	}
	a.VMs = vms
	a.StateMachine.Transition(state.StateVMSelector)
	return nil
}

func (a *App) LoadContainers(vmIndex int) ([]docker.ContainerInfo, error) {
	if vmIndex < 0 || vmIndex >= len(a.VMs) {
		return nil, fmt.Errorf("invalid vm index")
	}

	containers, err := a.DockerController.ListContainers(a.VMs[vmIndex])
	if err != nil {
		return nil, err
	}
	return containers, nil
}
