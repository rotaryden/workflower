package deploy

import (
	_ "embed"
	"fmt"
	"workflower/lib/templating"
)

//go:embed service.template
var serviceTemplate string

// ServiceConfig holds template values for systemd service
type ServiceConfig struct {
	Description      string
	User             string
	Group             string
	WorkingDirectory string
	ExecStart        string
	EnvFile          string
	ReadWritePaths   string
}

// GenerateServiceFile generates a systemd service file from template
func GenerateServiceFile(cfg *Config) (string, error) {
	remotePath := cfg.RemotePath()

	serviceConfig := ServiceConfig{
		Description:      cfg.ServiceDescription,
		User:             cfg.ServiceUser,
		Group:            cfg.ServiceGroup,
		WorkingDirectory: remotePath,
		ExecStart:        fmt.Sprintf("%s/%s", remotePath, cfg.AppName),
		EnvFile:          fmt.Sprintf("%s/.env", remotePath),
		ReadWritePaths:   fmt.Sprintf("%s/uploads", remotePath),
	}

	content, err := templating.Execute(serviceTemplate, serviceConfig, templating.Text)
	if err != nil {
		return "", fmt.Errorf("failed to generate service file: %w", err)
	}

	return content, nil
}
