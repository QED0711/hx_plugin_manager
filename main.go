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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// --- TUI Model ---

type model struct {
	textInput   textinput.Model
	suggestions []string
	suggestion  string
	choice      string
	quitting    bool
}

func initialModel(pluginNames []string) model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.Prompt = ":"
	ti.CharLimit = 156
	ti.Width = 50

	return model{
		textInput:   ti,
		suggestions: pluginNames,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			m.choice = m.textInput.Value()
			return m, tea.Quit

		case "tab":
			if m.suggestion != "" {
				m.textInput.SetValue(m.suggestion)
				m.textInput.SetCursor(len(m.suggestion))
			}
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)

	// Update autocomplete suggestion
	val := m.textInput.Value()
	m.suggestion = ""
	if val != "" {
		for _, s := range m.suggestions {
			if strings.HasPrefix(strings.ToLower(s), strings.ToLower(val)) {
				m.suggestion = s
				break
			}
		}
	}

	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Calculate the ghost text for the suggestion
	display := m.textInput.View()
	if m.suggestion != "" && len(m.suggestion) > len(m.textInput.Value()) {
		ghostText := m.suggestion[len(m.textInput.Value()):]
		display += lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(ghostText)
	}

	return display
}

// --- Logic Helpers ---

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
	args := InitArguments()

	data, err := os.ReadFile(args.ConfigFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}
	var cfg Config
	yaml.Unmarshal(data, &cfg)

	if args.ListCommands {
		for _, plugin := range cfg.Plugins {
			fmt.Println(plugin.Key+": ", plugin.Name)
		}
		return
	}

	// 1. Collect names for autocomplete
	var names []string
	for _, p := range cfg.Plugins {
		names = append(names, p.Key)
	}

	// 2. Run the Input TUI
	p := tea.NewProgram(initialModel(names))
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	m := finalModel.(model)
	if m.quitting || m.choice == "" {
		return
	}

	// Map the chosen Name back to the Key
	choices := strings.Fields(m.choice)
	choice := choices[0]
	var selectedPlugin *Plugin
	for _, p := range cfg.Plugins {
		if p.Key == choice {
			selectedPlugin = &p
			choice = p.Key
			break
		}
	}

	// 3. Port over context
	selection := getSelection()
	context := map[string]string{
		"buffer_name":          os.Getenv("HX_BUFFER_NAME"),
		"cursor_line":          os.Getenv("HX_CURSOR_LINE"),
		"cursor_column":        os.Getenv("HX_CURSOR_COLUMN"),
		"language":             os.Getenv("HX_LANGUAGE"),
		"selection_line_start": os.Getenv("HX_SELECTION_LINE_START"),
		"selection_line_end":   os.Getenv("HX_SELECTION_LINE_END"),
		"selection_raw":        selection,
	}

	for i, arg := range choices {
		context[fmt.Sprintf("ARG%d", i)] = arg
	}

	if choice == "CONFIG" {
		for k, v := range context {
			fmt.Println(k + "=" + v)
		}
		return
	}

	// focus previous pane
	exec.Command("zellij", "action", "toggle-floating-panes").Run()
	time.Sleep(50 * time.Millisecond)

	if selectedPlugin != nil {
		context["selection"] = selectedPlugin.EscapeContent(context["selection_raw"])

		for _, step := range selectedPlugin.Steps {
			if step.Wait != 0 {
				time.Sleep(time.Duration(step.Wait) * time.Millisecond)
			}

			if step.VariableKey != "" {
				cmdStr := ReplacePlaceholders(step.VarCmd, context)
				cmd := exec.Command("sh", "-c", cmdStr)
				var out bytes.Buffer
				cmd.Stdout = &out
				cmd.Run()
				context[step.VariableKey] = strings.TrimSpace(out.String())
			}

			if step.Run != "" {
				cmdStr := ReplacePlaceholders(step.Run, context)
				cmd := exec.Command("sh", "-c", cmdStr)
				if step.CaptureOutput {
					var out bytes.Buffer
					cmd.Stdout = &out
					cmd.Run()
					context["output"] = strings.TrimSpace(out.String())
				} else {
					cmd.Stdout = os.Stdout
					cmd.Stdin = os.Stdin
					cmd.Run()
				}
			}

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
