package main

// place this command in your helix `config.toml` file in the normal and selection sections:
// [ ':pipe-to tee ~/.config/helix/plugins/selection.txt', ':sh zellij run --floating --height 10%% --width 50%% -x 25%% -y 45%% -n "commands" --close-on-exit -- env HX_BUFFER_NAME=$PWD/%{buffer_name} HX_CURSOR_LINE=%{cursor_line} HX_CURSOR_COLUMN=%{cursor_column} HX_LANGUAGE=%{language} HX_SELECTION_LINE_START=%{selection_line_start} HX_SELECTION_LINE_END=%{selection_line_end} ~/.config/helix/plugins/go/command_runner/command_runner']

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func getSelection() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config/helix/plugins/selection.txt")

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func main() {
	// 1. Load Config
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config/helix/plugins/config.yaml")
	if len(os.Args) >= 2 {
		configPath = os.Args[1]
	}
	data, _ := os.ReadFile(configPath)
	var cfg Config
	yaml.Unmarshal(data, &cfg)

	// 2. Simple prompt (Replace with fzf or bubbletea later)
	fmt.Print(":")
	var choice string
	fmt.Scanln(&choice)

	// 3. Port over helix context via env variables

	// We specifically read the selection from a file to deal with escaping errors
	selection := getSelection()

	context := map[string]string{
		"buffer_name":          os.Getenv("HX_BUFFER_NAME"),
		"cursor_line":          os.Getenv("HX_CURSOR_LINE"),
		"cursor_column":        os.Getenv("HX_CURSOR_COLUMN"),
		"language":             os.Getenv("HX_LANGUAGE"),
		"selection_line_start": os.Getenv("HX_SELECTION_LINE_START"),
		"selection_line_end":   os.Getenv("HX_SELECTION_LINE_END"),
		"selection_raw":        string(selection),
	}

	if choice == "CONFIG" {
		for k, v := range context {
			fmt.Println(k + "=" + v)
		}
		return
	}

	// focus the previous pane to make sure the actions apply to that pane
	exec.Command("zellij", "action", "toggle-floating-panes").Run()
	time.Sleep(50 * time.Millisecond)

	var lastOutput string

	for _, plugin := range cfg.Plugins {

		// replace selection with escaped version if specified in plugin config
		context["selection"] = plugin.EscapeContent(context["selection_raw"])

		if plugin.Key == choice {
			for _, step := range plugin.Steps {

				// Wait
				if step.Wait != 0 {
					time.Sleep(time.Duration(step.Wait) * time.Millisecond)
				}

				// Store variable from command
				if step.VariableKey != "" {
					cmdStr := ReplacePlaceholders(step.VarCmd, context)
					cmd := exec.Command("sh", "-c", cmdStr)

					var out bytes.Buffer
					cmd.Stdout = &out
					cmd.Run()
					context[step.VariableKey] = strings.TrimSpace(out.String())
				}

				// Handle External Command
				if step.Run != "" {
					cmdStr := ReplacePlaceholders(step.Run, context)
					cmd := exec.Command("sh", "-c", cmdStr)

					if step.CaptureOutput {
						var out bytes.Buffer
						cmd.Stdout = &out
						cmd.Run()
						lastOutput = strings.TrimSpace(out.String())
						context["output"] = lastOutput // Update context for next steps
					} else {
						cmd.Stdout = os.Stdout
						cmd.Stdin = os.Stdin
						cmd.Run()
					}
				}

				// Handle Helix Interaction
				if step.HelixAction == "command" {
					SendHelixCommand("", ReplacePlaceholders(step.Command, context))
				} else if step.HelixAction == "keystrokes" {
					var finalSeq []string
					for _, s := range step.Sequence {
						finalSeq = append(finalSeq, ReplacePlaceholders(s, context))
					}
					SendKeys("", finalSeq)
				}
			}
		}
	}
}
