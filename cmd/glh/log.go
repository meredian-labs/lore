package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/store"
)

type gitCommit struct {
	SHA     string
	Subject string
	Date    string
}

func runLog(args []string) int {
	commits, err := gitLog(args)
	if err != nil {
		// git failed — fall back to plain passthrough.
		return gitExitCode(append([]string{"log"}, args...))
	}

	loreRoot, _ := findLoreRoot()

	var s *store.Store
	if loreRoot != "" {
		s, _ = store.Open(filepath.Join(loreRoot, "lore.db"))
		if s != nil {
			defer s.Close()
		}
	}

	ctx := context.Background()

	for _, c := range commits {
		short := c.SHA
		if len(short) > 7 {
			short = short[:7]
		}

		var blobLine string
		if s != nil {
			b, _ := s.BlobByCommitStart(ctx, c.SHA)
			if b != nil && b.Kind != blob.KindCheckpoint {
				blobLine = fmt.Sprintf("  ●  %s  [%s, %s]",
					truncate(b.Title, 40), string(b.Kind), trustLabel(b.TrustLevel),
				)
			}
		}

		subject := truncate(c.Subject, 50)
		if blobLine != "" {
			fmt.Fprintf(os.Stdout, "%s  %s  %-50s%s\n", short, c.Date, subject, blobLine)
		} else {
			fmt.Fprintf(os.Stdout, "%s  %s  %s\n", short, c.Date, subject)
		}
	}

	return 0
}

// gitLog runs git log with a machine-readable format and returns parsed commits.
// User-supplied args are appended so flags like --oneline or -n work as expected.
func gitLog(args []string) ([]gitCommit, error) {
	// If the user passed --format or --pretty, respect it by falling back to passthrough.
	for _, a := range args {
		if strings.HasPrefix(a, "--format") || strings.HasPrefix(a, "--pretty") {
			return nil, fmt.Errorf("custom format")
		}
	}

	glhArgs := []string{"log", "--format=%H|%s|%ad", "--date=short"}
	// Only set a default limit if the user hasn't passed -n / --max-count.
	hasLimit := false
	for _, a := range args {
		if a == "-n" || strings.HasPrefix(a, "--max-count") || strings.HasPrefix(a, "-n") {
			hasLimit = true
			break
		}
	}
	if !hasLimit {
		glhArgs = append(glhArgs, "-n", "40")
	}
	glhArgs = append(glhArgs, args...)

	out, err := exec.Command("git", glhArgs...).Output()
	if err != nil {
		return nil, err
	}

	var commits []gitCommit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		date, err := time.Parse("2006-01-02", parts[2])
		if err != nil {
			continue
		}
		commits = append(commits, gitCommit{
			SHA:     parts[0],
			Subject: parts[1],
			Date:    date.Format("2006-01-02"),
		})
	}
	return commits, nil
}
