package ssh

import (
	"fmt"
	"sync"
	"time"

	"tui-ssh-docker/internal/vm"

	"golang.org/x/crypto/ssh"
)

type SSHManager struct {
	mu      sync.Mutex
	clients map[string]*ssh.Client
}

func NewSSHManager() *SSHManager {
	return &SSHManager{
		clients: make(map[string]*ssh.Client),
	}
}

func (m *SSHManager) Connect(vm vm.VMConfig) (*ssh.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if existing client is still alive
	if client, ok := m.clients[vm.ID]; ok {
		if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err == nil {
			return client, nil
		}
		// If check fails, close and remove it
		client.Close()
		delete(m.clients, vm.ID)
	}

	config := &ssh.ClientConfig{
		User: vm.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(vm.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", vm.Host, vm.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh: %w", err)
	}

	m.clients[vm.ID] = client
	return client, nil
}

func (m *SSHManager) GetClient(vmID string) (*ssh.Client, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	client, ok := m.clients[vmID]
	return client, ok
}

func (m *SSHManager) Close(vmID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[vmID]; ok {
		client.Close()
		delete(m.clients, vmID)
	}
}

func (m *SSHManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, client := range m.clients {
		client.Close()
		delete(m.clients, id)
	}
}
