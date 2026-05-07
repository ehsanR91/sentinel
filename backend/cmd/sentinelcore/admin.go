package main

import (
	"fmt"
	"os"
	"strings"

	appauth "github.com/ehsanR91/sentinelcore/internal/auth"
	"github.com/ehsanR91/sentinelcore/internal/config"
	"github.com/ehsanR91/sentinelcore/internal/db"
)

// runAdminTool handles the "sentinelcore admin <command> [args]" subcommand.
// This is called by deploy-sentinel.sh for maintenance operations.
func runAdminTool(args []string) {
	if len(args) < 1 {
		printAdminUsage()
		os.Exit(1)
	}

	cfg := config.Load()
	if err := db.Init(cfg.DBPath, cfg.SecretsKeyPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: DB init failed: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "rotate-master-key":
		if cfg.SecretsKeyPath == "" {
			fmt.Fprintln(os.Stderr, "error: SECRETS_KEY_PATH is not configured")
			os.Exit(1)
		}
		if err := db.RotateSecretsKey(cfg.SecretsKeyPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: rotating master key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Master key rotated successfully: %s\n", cfg.SecretsKeyPath)

	case "reset-password":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: sentinelcore admin reset-password <username> <new-password>\n")
			os.Exit(1)
		}
		username, password := args[1], args[2]
		if len(password) < 12 {
			fmt.Fprintf(os.Stderr, "error: password must be at least 12 characters\n")
			os.Exit(1)
		}
		hash, err := appauth.HashPassword(password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: hashing password: %v\n", err)
			os.Exit(1)
		}
		if err := db.UpdatePassword(username, hash); err != nil {
			if err == db.ErrNotFound {
				fmt.Fprintf(os.Stderr, "error: user '%s' not found\n", username)
			} else {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}
		fmt.Printf("Password updated for user: %s\n", username)

	case "reset-2fa":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: sentinelcore admin reset-2fa <username>\n")
			os.Exit(1)
		}
		username := args[1]
		if err := db.ResetTOTP(username); err != nil {
			if err == db.ErrNotFound {
				fmt.Fprintf(os.Stderr, "error: user '%s' not found\n", username)
			} else {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}
		fmt.Printf("2FA disabled for user: %s\n", username)

	case "list-users":
		names, err := db.ListUsernames()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(names) == 0 {
			fmt.Println("(no users found)")
		} else {
			fmt.Println(strings.Join(names, "\n"))
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printAdminUsage()
		os.Exit(1)
	}
}

func printAdminUsage() {
	fmt.Fprintln(os.Stderr, "Usage: sentinelcore admin <command> [args]")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  rotate-master-key")
	fmt.Fprintln(os.Stderr, "  reset-password <username> <new-password>")
	fmt.Fprintln(os.Stderr, "  reset-2fa      <username>")
	fmt.Fprintln(os.Stderr, "  list-users")
}
