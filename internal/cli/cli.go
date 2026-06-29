package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	syncer "obsidian-preference-sync/internal/sync"
)

type checkFailedError struct{}

func (checkFailedError) Error() string { return "check failed: changes would be made" }

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var checkErr checkFailedError
	if errors.As(err, &checkErr) {
		return 1
	}
	return 2
}

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer, version string) error {
	fs := flag.NewFlagSet("obsidian-preference-sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "Print version and exit")
	vaultPath := fs.String("vault", "", "Path to an Obsidian vault")
	configPath := fs.String("config", "", "Path to config.toml")
	check := fs.Bool("check", false, "Show planned changes and exit with 1 if changes would be made")
	dryRun := fs.Bool("dry-run", false, "Show planned changes without writing")
	verbose := fs.Bool("verbose", false, "Print extra progress")
	allowDangerous := fs.Bool("allow-dangerous", false, "Allow syncing settings for dangerous plugins")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		if version == "" {
			version = "dev"
		}
		_, err := fmt.Fprintln(stdout, version)
		return err
	}
	if *vaultPath == "" {
		return fmt.Errorf("--vault is required")
	}
	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}

	planOnly := *check || *dryRun
	plan, cfg, v, err := syncer.BuildPlan(ctx, syncer.Options{
		VaultPath:      *vaultPath,
		ConfigPath:     *configPath,
		AllowDangerous: *allowDangerous,
	})
	if err != nil {
		return err
	}
	if planOnly {
		syncer.RenderPlan(plan, *verbose, stdout, stderr)
	}
	if !planOnly {
		if err := syncer.Apply(ctx, plan, cfg, v, *verbose, stdout); err != nil {
			return err
		}
	}

	if *check && plan.Changed() {
		return checkFailedError{}
	}
	return nil
}
