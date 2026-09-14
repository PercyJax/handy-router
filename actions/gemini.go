package actions

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

func OpenGemini(query string, urlTemplate string, browser string, browserArgs []string) error {
	encoded := url.QueryEscape(query)
	u := strings.Replace(urlTemplate, "{query}", encoded, 1)

	var cmd *exec.Cmd
	switch {
	case browser != "":
		args := make([]string, 0, len(browserArgs)+1)
		args = append(args, browserArgs...)
		args = append(args, u)
		cmd = exec.Command(browser, args...)
	case runtime.GOOS == "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}

	fmt.Printf("Opening Gemini: %v (%s)\n", cmd.Args, u)
	return cmd.Start()
}
