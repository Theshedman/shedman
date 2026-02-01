package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// SnapshotScheduleCmd is the command to manage scheduling
var SnapshotScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage snapshot scheduling",
}

var SnapshotScheduleEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable scheduled snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotScheduleEnable(engine, cmd.OutOrStdout())
	},
}

var SnapshotScheduleDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable scheduled snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotScheduleDisable(engine, cmd.OutOrStdout())
	},
}

var SnapshotScheduleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show scheduler status",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotScheduleStatus(engine, cmd.OutOrStdout())
	},
}

var SnapshotScheduleRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run scheduled snapshot now",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotScheduleRun(engine, cmd.OutOrStdout())
	},
}

func init() {
	SnapshotScheduleCmd.AddCommand(SnapshotScheduleEnableCmd)
	SnapshotScheduleCmd.AddCommand(SnapshotScheduleDisableCmd)
	SnapshotScheduleCmd.AddCommand(SnapshotScheduleStatusCmd)
	SnapshotScheduleCmd.AddCommand(SnapshotScheduleRunCmd)
}

func RunSnapshotScheduleEnable(engine *core.Engine, w io.Writer) error {
	sch := engine.GetScheduler()
	if sch == nil {
		return fmt.Errorf("scheduler not available")
	}
	if err := sch.Enable(); err != nil {
		return fmt.Errorf("failed to enable scheduler: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Snapshot scheduling enabled.")

	return nil
}

func RunSnapshotScheduleDisable(engine *core.Engine, w io.Writer) error {
	sch := engine.GetScheduler()
	if sch == nil {
		return fmt.Errorf("scheduler not available")
	}
	if err := sch.Disable(); err != nil {
		return fmt.Errorf("failed to disable scheduler: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Snapshot scheduling disabled.")

	return nil
}

func RunSnapshotScheduleStatus(engine *core.Engine, w io.Writer) error {
	sch := engine.GetScheduler()
	if sch == nil {
		return fmt.Errorf("scheduler not available")
	}
	status, err := sch.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	state := "disabled"
	if status.Enabled {
		state = "active"
	}
	_, _ = fmt.Fprintf(w, "Scheduler Status: %s\n", state)

	return nil
}

func RunSnapshotScheduleRun(engine *core.Engine, w io.Writer) error {
	sch := engine.GetScheduler()
	if sch == nil {
		return fmt.Errorf("scheduler not available")
	}
	if err := sch.RunNow(); err != nil {
		return fmt.Errorf("failed to run scheduler: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Snapshot scheduling triggered.")
	return nil
}
