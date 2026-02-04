package docker

import (
	"bufio"
	"fmt"
	"strings"

	"tui-ssh-docker/internal/ssh"
	"tui-ssh-docker/internal/vm"

	gossh "golang.org/x/crypto/ssh"
)

type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
}

type DockerController struct {
	sshManager *ssh.SSHManager
}

func NewDockerController(sshManager *ssh.SSHManager) *DockerController {
	return &DockerController{
		sshManager: sshManager,
	}
}

func (d *DockerController) ListContainers(vmConfig vm.VMConfig) ([]ContainerInfo, error) {
	client, err := d.sshManager.Connect(vmConfig)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Using --format to get clean output for parsing
	output, err := session.Output("docker ps --format '{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}'")
	if err != nil {
		return nil, fmt.Errorf("failed to run docker ps: %w", err)
	}

	var containers []ContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) >= 4 {
			containers = append(containers, ContainerInfo{
				ID:     parts[0],
				Name:   parts[1],
				Image:  parts[2],
				Status: parts[3],
			})
		}
	}

	return containers, nil
}

// GetExecSession is a helper to get an interactive session for docker exec
func (d *DockerController) GetExecSession(vmConfig vm.VMConfig, containerID string) (*gossh.Session, error) {
	client, err := d.sshManager.Connect(vmConfig)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}
