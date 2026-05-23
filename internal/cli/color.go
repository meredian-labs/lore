package cli

import (
	"fmt"
	"os"
)

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
)

func trustLabel(trustLevel int) string {
	switch trustLevel {
	case 1:
		return "[GroundTruth]"
	case 2:
		return "[AgentTruth]"
	case 3:
		return "[HumanAsserted]"
	case 4:
		return "[LoreInferred]"
	default:
		return fmt.Sprintf("[trust=%d]", trustLevel)
	}
}

func colorTrustLabel(trustLevel int) string {
	label := trustLabel(trustLevel)
	if !colorEnabled() {
		return label
	}
	switch trustLevel {
	case 2:
		return ansiGreen + label + ansiReset
	case 4:
		return ansiYellow + label + ansiReset
	default:
		return label
	}
}

func bold(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiBold + s + ansiReset
}

func dim(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiDim + s + ansiReset
}
