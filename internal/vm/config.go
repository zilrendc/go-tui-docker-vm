package vm

import (
	"encoding/json"
	"fmt"
	"os"
)

type VMConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type VMConfigFile struct {
	VMs []VMConfig `json:"vms"`
}

func LoadVMConfig(path string) ([]VMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configFile VMConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	for i, vm := range configFile.VMs {
		if vm.ID == "" {
			return nil, fmt.Errorf("vm at index %d is missing ID", i)
		}
		if vm.Name == "" {
			return nil, fmt.Errorf("vm %s is missing Name", vm.ID)
		}
		if vm.Host == "" {
			return nil, fmt.Errorf("vm %s is missing Host", vm.ID)
		}
		if vm.Port <= 0 {
			return nil, fmt.Errorf("vm %s has invalid Port: %d", vm.ID, vm.Port)
		}
		if vm.User == "" {
			return nil, fmt.Errorf("vm %s is missing User", vm.ID)
		}
		if vm.Password == "" {
			return nil, fmt.Errorf("vm %s is missing Password", vm.ID)
		}
	}

	return configFile.VMs, nil
}
