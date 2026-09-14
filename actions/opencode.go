package actions

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"

	"github.com/PercyJax/handy-router/opencode"
)

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		u, err := user.Current()
		if err == nil {
			return u.HomeDir + path[1:]
		}
	}
	return path
}

func OpenOpenCode(query string, terminal string, terminalArgs []string, binary string, extraArgs []string, serverURL string, dir string) error {
	dir = expandHome(dir)

	// Create a new session on the opencode serve instance
	sessionID, err := opencode.CreateSession(serverURL)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	fmt.Printf("Created opencode session: %s\n", sessionID)

	// Send the transcript as the first message (async)
	if err := opencode.SendMessage(serverURL, sessionID, query); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	fmt.Printf("Sent message to session %s: %q\n", sessionID, query)

	// Open TUI attached to the session
	args := make([]string, 0, len(terminalArgs)+2+3+len(extraArgs))
	args = append(args, terminalArgs...)
	args = append(args, binary, "attach", serverURL, "--session", sessionID)
	if dir != "" {
		args = append(args, "--dir", dir)
	}
	args = append(args, extraArgs...)

	cmd := exec.Command(terminal, args...)

	fmt.Printf("Opening OpenCode TUI in %s: %s attach %s --session %s\n", terminal, binary, serverURL, sessionID)
	return cmd.Start()
}
