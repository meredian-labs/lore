package glh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/store"
)

func runStatus(args []string) int {
	code := gitExitCode(append([]string{"status"}, args...))
	printLoreStatusFooter()
	return code
}

func printLoreStatusFooter() {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return
	}

	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return
	}
	defer s.Close()

	ctx := context.Background()

	sep := strings.Repeat("─", 68)
	fmt.Fprintf(os.Stdout, "\n%s\n", sep)

	pending, _ := s.PendingTaskCount(ctx)
	if pending > 0 {
		fmt.Fprintf(os.Stdout, "Pending tasks:  %d  (extraction on next commit)\n", pending)
	} else {
		fmt.Fprintf(os.Stdout, "Pending tasks:  0\n")
	}

	blobs, _ := s.BlobLog(ctx, 1)
	for _, b := range blobs {
		if b.Kind == blob.KindCheckpoint {
			continue
		}
		date := time.Unix(0, b.EndedAt).UTC().Format("2006-01-02")
		fmt.Fprintf(os.Stdout, "Last blob:      %-36s  %s  [%s, %s]\n",
			truncate(b.Title, 36), date, string(b.Kind), trustLabel(b.TrustLevel),
		)
		break
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func trustLabel(level int) string {
	switch level {
	case 1:
		return "GroundTruth"
	case 2:
		return "AgentTruth"
	case 3:
		return "HumanAssert"
	default:
		return "LoreInferred"
	}
}
