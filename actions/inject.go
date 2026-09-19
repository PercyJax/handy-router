package actions

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// TypeText injects text into the active window using a ydotool-compatible
// typing tool. It reads the text from stdin (`type -f -`) so no argument or
// shell escaping is needed, and it works in terminals, text boxes, and any
// other focused input across Wayland/X11/macOS/Windows.
// keyDelay sets the delay in milliseconds between keystrokes (0 uses ydotool's default of 20ms).
func TypeText(text, binary string, keyDelay int) error {
	if binary == "" {
		binary = "ydotool"
	}

	args := []string{"type", "-f", "-"}
	if keyDelay > 0 {
		args = append(args, "-d", strconv.Itoa(keyDelay), "-H", strconv.Itoa(keyDelay))
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdin = strings.NewReader(text)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s type: %w (%s)", binary, err, strings.TrimSpace(string(out)))
	}
	return nil
}
