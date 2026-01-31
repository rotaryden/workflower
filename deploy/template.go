package deploy

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed service.template
var serviceTemplate string

// ServiceConfig holds template values for systemd service
type ServiceConfig struct {
	Description      string
	User             string
	Group            string
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

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, serviceConfig); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
