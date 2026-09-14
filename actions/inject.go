package actions

import (
	"fmt"
	"os/exec"
	"strings"
)

// TypeText injects text into the active window using a ydotool-compatible
// typing tool. It reads the text from stdin (`type -f -`) so no argument or
// shell escaping is needed, and it works in terminals, text boxes, and any
// other focused input across Wayland/X11/macOS/Windows.
func TypeText(text, binary string) error {
	if binary == "" {
		binary = "ydotool"
	}

	cmd := exec.Command(binary, "type", "-f", "-")
	cmd.Stdin = strings.NewReader(text)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s type: %w (%s)", binary, err, strings.TrimSpace(string(out)))
	}
	return nil
}
