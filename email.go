package azureemail

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed azure_email_service/*
var content embed.FS

type EmailService struct {
	BaseURL string
	cmd     *exec.Cmd
}

func NewEmailService(connectionString, senderAddress string) (*EmailService, error) {
	service := &EmailService{
		BaseURL: "http://localhost:8005",
	}

	// Extract and run the Python service
	pythonDir, err := extractPythonService()
	if err != nil {
		return nil, fmt.Errorf("failed to extract Python service: %v", err)
	}

	cmd := exec.Command("python", filepath.Join(pythonDir, "azure_email_service/app.py"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"AZURE_CONNECTION_STRING="+connectionString,
		"SENDER_ADDRESS="+senderAddress,
	)

	// Start the Python service
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Python service: %v", err)
	}

	service.cmd = cmd

	// Wait for a few seconds to ensure the service is up
	time.Sleep(5 * time.Second)

	return service, nil
}

func (s *EmailService) SendEmail(email, subject, plainText, htmlContent string) (string, error) {
	url := fmt.Sprintf("%s/send-email", s.BaseURL)
	payload := map[string]string{
		"email":       email,
		"subject":     subject,
		"plainText":   plainText,
		"htmlContent": htmlContent,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %v", result["error"])
	}

	return result["message"].(string), nil
}

func (s *EmailService) Stop() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

func extractPythonService() (string, error) {
	dir, err := os.MkdirTemp("", "azure_email_service")
	if err != nil {
		return "", err
	}

	err = fs.WalkDir(content, "azure_email_service", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("azure_email_service", path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dir, relPath), 0755)
		}

		data, err := content.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(filepath.Join(dir, relPath), data, 0644)
	})

	return dir, err
}
