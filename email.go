package azureemail

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Embed the Python executable
//
//go:embed azure_email_service_executable/app
var content embed.FS

type EmailService struct {
	BaseURL string
	cmd     *exec.Cmd
}

func NewEmailService(connectionString, senderAddress string) (*EmailService, error) {
	service := &EmailService{
		BaseURL: "http://localhost:8005",
	}

	// Extract and run the Python executable
	exePath, err := extractExecutable()
	if err != nil {
		return nil, fmt.Errorf("failed to extract Python executable: %v", err)
	}

	cmd := exec.Command(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"AZURE_CONNECTION_STRING="+connectionString,
		"SENDER_ADDRESS="+senderAddress,
	)

	// Start the Python executable
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Python executable: %v", err)
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

func extractExecutable() (string, error) {
	dir, err := os.MkdirTemp("", "azure_email_service_executable")
	if err != nil {
		return "", err
	}

	exePath := filepath.Join(dir, "app")
	data, err := content.ReadFile("azure_email_service_executable/app")
	if err != nil {
		return "", err
	}

	err = os.WriteFile(exePath, data, 0755)
	if err != nil {
		return "", err
	}

	return exePath, nil
}
