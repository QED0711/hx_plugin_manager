package main

import (
	"strings"
)

type Step struct {
	Run           string   `yaml:"run,omitempty"`
	CaptureOutput bool     `yaml:"capture_output,omitempty"`
	Interactive   bool     `yaml:"interactive,omitempty"`
	HelixAction   string   `yaml:"helix_action,omitempty"` // "command" or "keystrokes"
	Command       string   `yaml:"command,omitempty"`
	Sequence      []string `yaml:"sequence,omitempty"`
}

type Plugin struct {
	Name        string   `yaml:"name"`
	Key         string   `yaml:"key"`
	EscapeChars []string `yaml:"escape_chars,omitempty"`
	Steps       []Step   `yaml:"steps"`
}

type Config struct {
	Plugins []Plugin `yaml:"plugins"`
}

func (p *Plugin) EscapeContent(input string) string {
	if len(p.EscapeChars) == 0 {
		return input
	}

	escaped := input
	for _, char := range p.EscapeChars {
		if char == "" {
			continue
		}
		// Replace every instance of 'char' with '\char'
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}
	return escaped
}

// ReplacePlaceholders swaps {file}, {selection}, etc., with real data
func ReplacePlaceholders(input string, context map[string]string) string {
	for k, v := range context {
		input = strings.ReplaceAll(input, "{"+k+"}", v)
	}
	return input
}
