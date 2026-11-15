package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// LoadSystemPrompt reads the required server prompt once at startup.
func (c Buddy) LoadSystemPrompt() (string, error) {
	if strings.TrimSpace(c.SystemPromptFile) == "" {
		return "", errors.New("SYSTEM_PROMPT_FILE is required for serve")
	}

	data, err := os.ReadFile(c.SystemPromptFile)
	if err != nil {
		return "", fmt.Errorf("read SYSTEM_PROMPT_FILE: %w", err)
	}

	if strings.TrimSpace(string(data)) == "" {
		return "", errors.New("SYSTEM_PROMPT_FILE must contain a nonempty prompt")
	}

	return string(data), nil
}
