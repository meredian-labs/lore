package glh

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runCommit(args []string) int {
	recap := false
	filtered := args[:0]
	for _, a := range args {
		if a == "--recap" {
			recap = true
		} else {
			filtered = append(filtered, a)
		}
	}

	code := gitExitCode(append([]string{"commit"}, filtered...))
	if code != 0 {
		return code
	}

	if !hooksInstalled() {
		fireLoreHook("commit")
	}

	if recap {
		runRecap()
	}

	return 0
}

func runRecap() {
	fmt.Fprint(os.Stderr, "Intent (one line, Enter to skip): ")
	var line string
	fmt.Scanln(&line)
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	payload := map[string]string{"summary": line}
	data, _ := json.Marshal(payload)

	cmd := exec.Command("lore", "hook", "agent-recap", "human:glh")
	cmd.Stdin = strings.NewReader(string(data))
	cmd.Stderr = os.Stderr
	cmd.Run()
}
