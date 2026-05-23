package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nishchay/lore/internal/store"
	"github.com/nishchay/lore/internal/task"
	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record <note>",
	Short: "Emit a Note task (developer annotation)",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRecord,
}

func runRecord(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	note := strings.Join(args, " ")
	if err := recordNote(cmd.Context(), loreRoot, note); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Recorded: %s\n", note)
	return nil
}

func recordNote(ctx context.Context, loreRoot, note string) error {
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()
	return s.InsertTask(ctx, task.Task{
		ID:         uuid.NewString(),
		Kind:       task.KindNote,
		Detail:     note,
		Source:     "human",
		TrustLevel: 1,
		TS:         time.Now().UnixNano(),
	})
}
