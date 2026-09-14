package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var hyperdriveCmd = &cobra.Command{
	Use:   "hyperdrive",
	Short: "Manage Cloudflare Hyperdrive configs",
	Long: `Hyperdrive configuration management for accelerated database connections from Workers.

Commands:
  create    Create a Hyperdrive config
  list      List all Hyperdrive configs
  get       Get config details
  update    Update a Hyperdrive config
  delete    Delete a Hyperdrive config

Hyperdrive accelerates access to existing databases from Cloudflare Workers,
making it faster to read and write data.

Examples:
  cosmoflare hyperdrive create my-db --origin-host=db.example.com --origin-port=5432 --origin-scheme=postgres --database=mydb --user=admin --password=secret
  cosmoflare hyperdrive list --json
  cosmoflare hyperdrive get CONFIG_ID
  cosmoflare hyperdrive update CONFIG_ID --name=new-name --origin-host=db2.example.com --origin-port=5432 --origin-scheme=postgres --database=mydb --user=admin --password=secret
  cosmoflare hyperdrive delete CONFIG_ID --force`,
}

var (
	hdOriginHost   string
	hdOriginPort   int
	hdOriginScheme string
	hdDatabase     string
	hdUser         string
	hdPassword     string
	hdForce        bool
	hdName         string
)

var hyperdriveCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a Hyperdrive config",
	Long: `Create a new Hyperdrive configuration for accelerated database connections.

Requires origin database connection details: host, port, scheme, database name,
user, and password.

Supported schemes: postgres, postgresql.

Examples:
  cosmoflare hyperdrive create my-db --origin-host=db.example.com --origin-port=5432 --origin-scheme=postgres --database=mydb --user=admin --password=secret
  cosmoflare hyperdrive create staging-db --origin-host=staging.example.com --origin-port=5432 --origin-scheme=postgres --database=staging --user=reader --password=pass123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runHyperdriveCreate,
}

var hyperdriveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Hyperdrive configs",
	Long: `List all Hyperdrive configurations in the current account.

Examples:
  cosmoflare hyperdrive list
  cosmoflare hyperdrive list --json`,
	RunE: runHyperdriveList,
}

var hyperdriveGetCmd = &cobra.Command{
	Use:   "get [config-id]",
	Short: "Get Hyperdrive config details",
	Long: `Get details of a single Hyperdrive configuration by its ID.

Examples:
  cosmoflare hyperdrive get CONFIG_ID
  cosmoflare hyperdrive get CONFIG_ID --json`,
	Args: cobra.ExactArgs(1),
	RunE: runHyperdriveGet,
}

var hyperdriveUpdateCmd = &cobra.Command{
	Use:   "update [config-id]",
	Short: "Update a Hyperdrive config",
	Long: `Update an existing Hyperdrive configuration.

All origin fields are required for update (the API replaces the entire config).

Examples:
  cosmoflare hyperdrive update CONFIG_ID --name=new-name --origin-host=db.example.com --origin-port=5432 --origin-scheme=postgres --database=mydb --user=admin --password=secret
  cosmoflare hyperdrive update CONFIG_ID --name=renamed --origin-host=db2.example.com --origin-port=5432 --origin-scheme=postgres --database=newdb --user=admin --password=newsecret --json`,
	Args: cobra.ExactArgs(1),
	RunE: runHyperdriveUpdate,
}

var hyperdriveDeleteCmd = &cobra.Command{
	Use:   "delete [config-id]",
	Short: "Delete a Hyperdrive config",
	Long: `Delete a Hyperdrive configuration.

WARNING: This action is irreversible. Workers using this config will lose
accelerated database access.

Examples:
  cosmoflare hyperdrive delete CONFIG_ID
  cosmoflare hyperdrive delete CONFIG_ID --force`,
	Args: cobra.ExactArgs(1),
	RunE: runHyperdriveDelete,
}

func init() {
	rootCmd.AddCommand(hyperdriveCmd)

	hyperdriveCmd.AddCommand(hyperdriveCreateCmd)
	hyperdriveCmd.AddCommand(hyperdriveListCmd)
	hyperdriveCmd.AddCommand(hyperdriveGetCmd)
	hyperdriveCmd.AddCommand(hyperdriveUpdateCmd)
	hyperdriveCmd.AddCommand(hyperdriveDeleteCmd)

	// Create flags
	hyperdriveCreateCmd.Flags().StringVar(&hdOriginHost, "origin-host", "", "Origin database hostname")
	hyperdriveCreateCmd.Flags().IntVar(&hdOriginPort, "origin-port", 5432, "Origin database port")
	hyperdriveCreateCmd.Flags().StringVar(&hdOriginScheme, "origin-scheme", "postgres", "Origin database scheme (postgres, postgresql)")
	hyperdriveCreateCmd.Flags().StringVar(&hdDatabase, "database", "", "Origin database name")
	hyperdriveCreateCmd.Flags().StringVar(&hdUser, "user", "", "Origin database user")
	hyperdriveCreateCmd.Flags().StringVar(&hdPassword, "password", "", "Origin database password")
	_ = hyperdriveCreateCmd.MarkFlagRequired("origin-host")
	_ = hyperdriveCreateCmd.MarkFlagRequired("database")
	_ = hyperdriveCreateCmd.MarkFlagRequired("user")
	_ = hyperdriveCreateCmd.MarkFlagRequired("password")

	// Update flags
	hyperdriveUpdateCmd.Flags().StringVar(&hdName, "name", "", "New name for the config")
	hyperdriveUpdateCmd.Flags().StringVar(&hdOriginHost, "origin-host", "", "Origin database hostname")
	hyperdriveUpdateCmd.Flags().IntVar(&hdOriginPort, "origin-port", 5432, "Origin database port")
	hyperdriveUpdateCmd.Flags().StringVar(&hdOriginScheme, "origin-scheme", "postgres", "Origin database scheme (postgres, postgresql)")
	hyperdriveUpdateCmd.Flags().StringVar(&hdDatabase, "database", "", "Origin database name")
	hyperdriveUpdateCmd.Flags().StringVar(&hdUser, "user", "", "Origin database user")
	hyperdriveUpdateCmd.Flags().StringVar(&hdPassword, "password", "", "Origin database password")
	_ = hyperdriveUpdateCmd.MarkFlagRequired("origin-host")
	_ = hyperdriveUpdateCmd.MarkFlagRequired("database")
	_ = hyperdriveUpdateCmd.MarkFlagRequired("user")
	_ = hyperdriveUpdateCmd.MarkFlagRequired("password")

	// Delete flags
	hyperdriveDeleteCmd.Flags().BoolVar(&hdForce, "force", false, "Skip confirmation prompt")
}

func getHyperdriveService() (*cosmoflare.HyperdriveService, error) {
	return cosmoflare.NewHyperdriveServiceFromCreds(AccountID, APIToken)
}

func runHyperdriveCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	origin := cosmoflare.HyperdriveOriginConfig{
		Host:     hdOriginHost,
		Port:     hdOriginPort,
		Scheme:   hdOriginScheme,
		Database: hdDatabase,
		User:     hdUser,
		Password: hdPassword,
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create Hyperdrive config", map[string]interface{}{
				"name":        name,
				"origin_host": origin.Host,
				"origin_port": origin.Port,
				"scheme":      origin.Scheme,
				"database":    origin.Database,
				"user":        origin.User,
			})
		}
		printInfo("DRY RUN: Would create Hyperdrive config '%s' (host=%s, db=%s)", name, origin.Host, origin.Database)
		return nil
	}

	svc, err := getHyperdriveService()
	if err != nil {
		return fmt.Errorf("failed to create Hyperdrive service: %w", err)
	}

	cfg, err := svc.Create(context.Background(), name, origin)
	if err != nil {
		return outErr("failed to create Hyperdrive config", err)
	}

	if JSONOutput {
		return printSuccessJSON("Hyperdrive config created successfully", cfg)
	}

	printSuccess("Hyperdrive config '%s' created (ID: %s)", cfg.Name, cfg.ID)
	printInfo("Origin: %s://%s@%s:%d/%s", cfg.Origin.Scheme, cfg.Origin.User, cfg.Origin.Host, cfg.Origin.Port, cfg.Origin.Database)
	return nil
}

func runHyperdriveList(cmd *cobra.Command, args []string) error {
	svc, err := getHyperdriveService()
	if err != nil {
		return fmt.Errorf("failed to create Hyperdrive service: %w", err)
	}

	configs, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list Hyperdrive configs", err)
	}

	if JSONOutput {
		return printJSON(configs)
	}

	if len(configs) == 0 {
		printInfo("No Hyperdrive configs found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tHOST\tPORT\tDATABASE\tSCHEME")
	for _, c := range configs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			c.ID, c.Name, c.Origin.Host, c.Origin.Port, c.Origin.Database, c.Origin.Scheme)
	}
	w.Flush()
	printInfo("Total: %d config(s)", len(configs))
	return nil
}

func runHyperdriveGet(cmd *cobra.Command, args []string) error {
	configID := args[0]

	svc, err := getHyperdriveService()
	if err != nil {
		return fmt.Errorf("failed to create Hyperdrive service: %w", err)
	}

	cfg, err := svc.Get(context.Background(), configID)
	if err != nil {
		return outErr("failed to get Hyperdrive config", err)
	}

	if JSONOutput {
		return printJSON(cfg)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%s\n", cfg.ID)
	fmt.Fprintf(w, "Name:\t%s\n", cfg.Name)
	fmt.Fprintf(w, "Host:\t%s\n", cfg.Origin.Host)
	fmt.Fprintf(w, "Port:\t%d\n", cfg.Origin.Port)
	fmt.Fprintf(w, "Scheme:\t%s\n", cfg.Origin.Scheme)
	fmt.Fprintf(w, "Database:\t%s\n", cfg.Origin.Database)
	fmt.Fprintf(w, "User:\t%s\n", cfg.Origin.User)
	if cfg.Caching.Disabled != nil {
		fmt.Fprintf(w, "Caching Disabled:\t%v\n", *cfg.Caching.Disabled)
	}
	if cfg.Caching.MaxAge > 0 {
		fmt.Fprintf(w, "Cache Max Age:\t%d\n", cfg.Caching.MaxAge)
	}
	if cfg.Caching.StaleWhileRevalidate > 0 {
		fmt.Fprintf(w, "Stale While Revalidate:\t%d\n", cfg.Caching.StaleWhileRevalidate)
	}
	w.Flush()
	return nil
}

func runHyperdriveUpdate(cmd *cobra.Command, args []string) error {
	configID := args[0]

	params := cosmoflare.HyperdriveUpdateParams{
		Name: hdName,
		Origin: cosmoflare.HyperdriveOriginConfig{
			Host:     hdOriginHost,
			Port:     hdOriginPort,
			Scheme:   hdOriginScheme,
			Database: hdDatabase,
			User:     hdUser,
			Password: hdPassword,
		},
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update Hyperdrive config", map[string]interface{}{
				"config_id":   configID,
				"name":        hdName,
				"origin_host": hdOriginHost,
				"database":    hdDatabase,
			})
		}
		printInfo("DRY RUN: Would update Hyperdrive config '%s'", configID)
		return nil
	}

	svc, err := getHyperdriveService()
	if err != nil {
		return fmt.Errorf("failed to create Hyperdrive service: %w", err)
	}

	cfg, err := svc.Update(context.Background(), configID, params)
	if err != nil {
		return outErr("failed to update Hyperdrive config", err)
	}

	if JSONOutput {
		return printSuccessJSON("Hyperdrive config updated successfully", cfg)
	}

	printSuccess("Hyperdrive config '%s' updated (ID: %s)", cfg.Name, cfg.ID)
	printInfo("Origin: %s://%s@%s:%d/%s", cfg.Origin.Scheme, cfg.Origin.User, cfg.Origin.Host, cfg.Origin.Port, cfg.Origin.Database)
	return nil
}

func runHyperdriveDelete(cmd *cobra.Command, args []string) error {
	configID := args[0]

	if !hdForce && !DryRun {
		fmt.Printf("Are you sure you want to delete Hyperdrive config '%s'? [y/N]: ", configID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Hyperdrive config deletion cancelled")
			return nil
		}
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete Hyperdrive config", map[string]string{
				"config_id": configID,
			})
		}
		printInfo("DRY RUN: Would delete Hyperdrive config '%s'", configID)
		return nil
	}

	svc, err := getHyperdriveService()
	if err != nil {
		return fmt.Errorf("failed to create Hyperdrive service: %w", err)
	}

	if err := svc.Delete(context.Background(), configID); err != nil {
		return outErr("failed to delete Hyperdrive config", err)
	}

	if JSONOutput {
		return printSuccessJSON("Hyperdrive config deleted successfully", map[string]string{
			"config_id": configID,
		})
	}
	printSuccess("Hyperdrive config '%s' deleted successfully!", configID)
	return nil
}
