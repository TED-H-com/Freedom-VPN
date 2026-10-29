// Command fvpn-usage-admin provides operator CLI commands for usage ledger recovery
// (freeze, begin-recovery, complete-recovery), operational adjustments, and member
// drill-down (§15.4, §17.3). Every mutating command records an append-only audit row.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
)

func main() {
	ctx := context.Background()
	if err := runMain(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "fvpn-usage-admin: %v\n", err)
		os.Exit(1)
	}
}

func runMain(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("fvpn-usage-admin", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configFile := fs.String("config", "", "central JSON configuration file")
	usageConfigFile := fs.String("usage-config", "", "optional usage JSON configuration file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	c, err := config.Load(*configFile)
	if err != nil {
		return err
	}
	if err := config.ValidateUsageCentralBounds(c); err != nil {
		return err
	}
	usagePath := c.UsageConfigFile
	if *usageConfigFile != "" {
		usagePath = *usageConfigFile
	}
	uCfg, err := config.LoadUsage(usagePath)
	if err != nil {
		return err
	}

	dsn, err := config.ReadSecret(c.DatabaseURLFile)
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, dsn, db.OpenOptions{})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	prov, err := signing.OpenFileProvider(uCfg.SigningConfigFile)
	if err != nil {
		return &config.SettingError{Name: "signing_config_file", Err: err}
	}

	ps := usage.NewProviderSigner(prov)
	svc, err := usage.NewService(usage.Config{
		Pool:              pool,
		Signer:            ps,
		LedgerEpochSigner: ps,
	})
	if err != nil {
		return err
	}
	return Run(ctx, svc, fs.Args(), stdout)
}

// Run executes an operator subcommand against the given usage.Service and writes JSON to stdout.
func Run(ctx context.Context, svc *usage.Service, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("subcommand required: freeze | begin-recovery | complete-recovery | adjust | drill-down")
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "freeze":
		fs := flag.NewFlagSet("freeze", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		actor := fs.String("actor", "", "actor UUID or identifier")
		reason := fs.String("reason", "", "non-blank freeze reason")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if err := svc.FreezeIngest(ctx, *actor, *reason); err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{
			"status": "frozen",
			"actor":  *actor,
			"reason": *reason,
		})

	case "begin-recovery":
		fs := flag.NewFlagSet("begin-recovery", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		actor := fs.String("actor", "", "actor UUID or identifier")
		reason := fs.String("reason", "", "non-blank recovery reason")
		newEpochStr := fs.String("new-epoch", "", "optional UUID for new ledger_epoch")
		restoredToStr := fs.String("restored-to", "", "optional RFC3339 timestamp of restored DB snapshot")
		affectedSinceStr := fs.String("affected-since", "", "optional RFC3339 timestamp of earliest affected window")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		var req usage.RecoveryRequest
		req.Actor = *actor
		req.Reason = *reason
		if *newEpochStr != "" {
			id, err := uuid.Parse(*newEpochStr)
			if err != nil {
				return fmt.Errorf("invalid --new-epoch: %w", err)
			}
			req.NewLedgerEpoch = id
		}
		if *restoredToStr != "" {
			t, err := time.Parse(time.RFC3339, *restoredToStr)
			if err != nil {
				return fmt.Errorf("invalid --restored-to: %w", err)
			}
			req.RestoredTo = t.UTC()
		}
		if *affectedSinceStr != "" {
			t, err := time.Parse(time.RFC3339, *affectedSinceStr)
			if err != nil {
				return fmt.Errorf("invalid --affected-since: %w", err)
			}
			req.AffectedSince = t.UTC()
		}
		state, err := svc.BeginRecovery(ctx, req)
		if err != nil {
			return err
		}
		return writeJSON(stdout, state)

	case "complete-recovery":
		fs := flag.NewFlagSet("complete-recovery", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		actor := fs.String("actor", "", "actor UUID or identifier")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if err := svc.CompleteRecovery(ctx, *actor); err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{
			"status": "normal",
			"actor":  *actor,
		})

	case "adjust":
		fs := flag.NewFlagSet("adjust", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		memberIDStr := fs.String("member-id", "", "member UUID")
		period := fs.String("period", "", "UTC month YYYY-MM")
		upBytes := fs.Int64("upload-bytes", 0, "signed upload byte delta")
		downBytes := fs.Int64("download-bytes", 0, "signed download byte delta")
		actor := fs.String("actor", "", "actor UUID or identifier")
		reason := fs.String("reason", "", "non-blank adjustment reason")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		memberID, err := uuid.Parse(*memberIDStr)
		if err != nil {
			return fmt.Errorf("invalid --member-id: %w", err)
		}
		adj, err := svc.CreateAdjustment(ctx, usage.AdjustmentCreate{
			MemberID:      memberID,
			Period:        *period,
			UploadBytes:   *upBytes,
			DownloadBytes: *downBytes,
			Reason:        *reason,
			Actor:         *actor,
		})
		if err != nil {
			return err
		}
		return writeJSON(stdout, adj)

	case "drill-down":
		fs := flag.NewFlagSet("drill-down", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		memberIDStr := fs.String("member-id", "", "member UUID")
		period := fs.String("period", "", "UTC month YYYY-MM")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		memberID, err := uuid.Parse(*memberIDStr)
		if err != nil {
			return fmt.Errorf("invalid --member-id: %w", err)
		}
		dd, err := svc.GetMemberDrillDown(ctx, memberID, *period)
		if err != nil {
			return err
		}
		return writeJSON(stdout, dd)

	default:
		return fmt.Errorf("unknown subcommand %q (expected freeze | begin-recovery | complete-recovery | adjust | drill-down)", sub)
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
