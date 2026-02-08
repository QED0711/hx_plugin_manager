package main

import (
	"os/exec"
)

// SendKeys translates special strings like +esc+ into bytes for Zellij
func SendKeys(paneID string, keys []string) error {
	for _, key := range keys {
		var cmd *exec.Cmd
		switch key {
		case "<esc>":
			cmd = exec.Command("zellij", "action", "write", "27")
		case "<enter>":
			cmd = exec.Command("zellij", "action", "write", "13")
		default:
			// Write literal string
			cmd = exec.Command("zellij", "action", "write-chars", key)
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func SendHelixCommand(paneID string, command string) error {
	// 1. Send ':' to open command bar
	exec.Command("zellij", "action", "write", "58").Run()
	// 2. Type the command
	exec.Command("zellij", "action", "write-chars", command).Run()
	// 3. Press Enter
	return exec.Command("zellij", "action", "write", "13").Run()
}
