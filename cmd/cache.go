package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage Cloudflare Cache",
	Long: `Cache management operations for Cloudflare zones.

Cache is zone-scoped — operations require a zone ID, not an account ID.

Commands:
  purge       Purge cached content (all, by URL, tag, or host)
  settings    View or update cache settings

Examples:
  cosmoflare cache purge abc123 --all --force
  cosmoflare cache purge abc123 --url=https://example.com/style.css
  cosmoflare cache purge abc123 --tag=static --tag=images
  cosmoflare cache purge abc123 --host=assets.example.com
  cosmoflare cache settings abc123
  cosmoflare cache settings abc123 --browser-ttl=3600 --cache-level=aggressive`,
}

var (
	cachePurgeAll    bool
	cachePurgeForce  bool
	cachePurgeURLs   []string
	cachePurgeTags   []string
	cachePurgeHosts  []string
	cacheBrowserTTL  int
	cacheDevMode     bool
	cacheCacheLevel  string
)

var cachePurgeCmd = &cobra.Command{
	Use:   "purge [zone-id]",
	Short: "Purge cached content",
	Long: `Purge cached content for a Cloudflare zone.

Supports purging by:
  --all           Purge everything (requires --force)
  --url           Purge specific URLs (up to 30)
  --tag           Purge by cache tag (Enterprise only)
  --host          Purge by hostname

At least one purge method must be specified.

Examples:
  # Purge everything (destructive — requires --force)
  cosmoflare cache purge abc123 --all --force

  # Purge specific URLs
  cosmoflare cache purge abc123 --url=https://example.com/style.css --url=https://example.com/app.js

  # Purge by cache tag (Enterprise)
  cosmoflare cache purge abc123 --tag=static --tag=images

  # Purge by hostname
  cosmoflare cache purge abc123 --host=assets.example.com --host=cdn.example.com

  # JSON output
  cosmoflare cache purge abc123 --all --force --json`,
	RunE: runCachePurge,
}

var cacheSettingsCmd = &cobra.Command{
	Use:   "settings [zone-id]",
	Short: "View or update cache settings",
	Long: `View or update cache-related settings for a Cloudflare zone.

When called without update flags, displays current settings.
When called with update flags, modifies the specified settings.

Update flags:
  --browser-ttl   Browser cache TTL in seconds (0 = respect origin headers)
  --dev-mode       Enable development mode (bypasses cache)
  --cache-level   Cache level: "aggressive", "basic", or "simplified"

Examples:
  # View current settings
  cosmoflare cache settings abc123
  cosmoflare cache settings abc123 --json

  # Update browser cache TTL
  cosmoflare cache settings abc123 --browser-ttl=3600

  # Enable development mode
  cosmoflare cache settings abc123 --dev-mode

  # Set aggressive caching
  cosmoflare cache settings abc123 --cache-level=aggressive

  # Multiple updates at once
  cosmoflare cache settings abc123 --browser-ttl=7200 --cache-level=aggressive`,
	RunE: runCacheSettings,
}

func init() {
	rootCmd.AddCommand(cacheCmd)

	cacheCmd.AddCommand(cachePurgeCmd)
	cacheCmd.AddCommand(cacheSettingsCmd)

	cachePurgeCmd.Flags().BoolVar(&cachePurgeAll, "all", false, "Purge all cached content")
	cachePurgeCmd.Flags().BoolVar(&cachePurgeForce, "force", false, "Confirm destructive purge-all operation")
	cachePurgeCmd.Flags().StringSliceVar(&cachePurgeURLs, "url", []string{}, "URLs to purge (up to 30)")
	cachePurgeCmd.Flags().StringSliceVar(&cachePurgeTags, "tag", []string{}, "Cache tags to purge (Enterprise only)")
	cachePurgeCmd.Flags().StringSliceVar(&cachePurgeHosts, "host", []string{}, "Hostnames to purge")

	cacheSettingsCmd.Flags().IntVar(&cacheBrowserTTL, "browser-ttl", 0, "Browser cache TTL in seconds")
	cacheSettingsCmd.Flags().BoolVar(&cacheDevMode, "dev-mode", false, "Enable development mode")
	cacheSettingsCmd.Flags().StringVar(&cacheCacheLevel, "cache-level", "", "Cache level (aggressive, basic, simplified)")
}

func getCacheService(zoneID string) (*cosmoflare.CacheService, error) {
	return cosmoflare.NewCacheServiceFromCreds(zoneID, APIToken)
}

func runCachePurge(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	// Validate that at least one purge method is specified
	if !cachePurgeAll && len(cachePurgeURLs) == 0 && len(cachePurgeTags) == 0 && len(cachePurgeHosts) == 0 {
		return fmt.Errorf("at least one purge method is required (--all, --url, --tag, or --host)")
	}

	// --all requires --force
	if cachePurgeAll && !cachePurgeForce && !DryRun {
		return fmt.Errorf("--all purge is destructive and requires --force flag\n\nUsage: cosmoflare cache purge %s --all --force", zoneID)
	}

	svc, err := getCacheService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create cache service: %w", err)
	}

	ctx := context.Background()

	// Purge all
	if cachePurgeAll {
		if DryRun {
			if JSONOutput {
				return printSuccessJSON("DRY RUN: Would purge all cache", map[string]string{"zone_id": zoneID})
			}
			printInfo("DRY RUN: Would purge all cached content for zone '%s'", zoneID)
			return nil
		}

		result, err := svc.PurgeAll(ctx)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to purge cache: %v", err))
			}
			return fmt.Errorf("failed to purge cache: %w", err)
		}

		if JSONOutput {
			return printSuccessJSON("Cache purged successfully", result)
		}
		printSuccess("All cached content purged for zone '%s'", zoneID)
		return nil
	}

	// Purge by URLs
	if len(cachePurgeURLs) > 0 {
		if DryRun {
			if JSONOutput {
				return printSuccessJSON("DRY RUN: Would purge URLs", map[string]interface{}{"zone_id": zoneID, "urls": cachePurgeURLs})
			}
			printInfo("DRY RUN: Would purge %d URL(s) from zone '%s'", len(cachePurgeURLs), zoneID)
			return nil
		}

		result, err := svc.PurgeByURLs(ctx, cachePurgeURLs)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to purge URLs: %v", err))
			}
			return fmt.Errorf("failed to purge URLs: %w", err)
		}

		if JSONOutput {
			return printSuccessJSON("URLs purged successfully", result)
		}
		printSuccess("Purged %d URL(s) from zone '%s'", len(cachePurgeURLs), zoneID)
		return nil
	}

	// Purge by tags
	if len(cachePurgeTags) > 0 {
		if DryRun {
			if JSONOutput {
				return printSuccessJSON("DRY RUN: Would purge tags", map[string]interface{}{"zone_id": zoneID, "tags": cachePurgeTags})
			}
			printInfo("DRY RUN: Would purge %d tag(s) from zone '%s'", len(cachePurgeTags), zoneID)
			return nil
		}

		result, err := svc.PurgeByTags(ctx, cachePurgeTags)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to purge tags: %v", err))
			}
			return fmt.Errorf("failed to purge tags: %w", err)
		}

		if JSONOutput {
			return printSuccessJSON("Tags purged successfully", result)
		}
		printSuccess("Purged %d tag(s) from zone '%s'", len(cachePurgeTags), zoneID)
		return nil
	}

	// Purge by hosts
	if len(cachePurgeHosts) > 0 {
		if DryRun {
			if JSONOutput {
				return printSuccessJSON("DRY RUN: Would purge hosts", map[string]interface{}{"zone_id": zoneID, "hosts": cachePurgeHosts})
			}
			printInfo("DRY RUN: Would purge %d host(s) from zone '%s'", len(cachePurgeHosts), zoneID)
			return nil
		}

		result, err := svc.PurgeByHosts(ctx, cachePurgeHosts)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to purge hosts: %v", err))
			}
			return fmt.Errorf("failed to purge hosts: %w", err)
		}

		if JSONOutput {
			return printSuccessJSON("Hosts purged successfully", result)
		}
		printSuccess("Purged %d host(s) from zone '%s'", len(cachePurgeHosts), zoneID)
		return nil
	}

	return nil
}

func runCacheSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getCacheService(zoneID)
	if err != nil {
		return fmt.Errorf("failed to create cache service: %w", err)
	}

	ctx := context.Background()

	// Check if any update flags were provided
	hasUpdates := cmd.Flags().Changed("browser-ttl") || cmd.Flags().Changed("dev-mode") || cmd.Flags().Changed("cache-level")

	if hasUpdates {
		var opts []cosmoflare.CacheOption
		if cmd.Flags().Changed("browser-ttl") {
			opts = append(opts, cosmoflare.WithBrowserCacheTTL(cacheBrowserTTL))
		}
		if cmd.Flags().Changed("dev-mode") {
			opts = append(opts, cosmoflare.WithDevMode(cacheDevMode))
		}
		if cmd.Flags().Changed("cache-level") {
			opts = append(opts, cosmoflare.WithCacheLevel(cacheCacheLevel))
		}

		if DryRun {
			if JSONOutput {
				return printSuccessJSON("DRY RUN: Would update cache settings", map[string]string{"zone_id": zoneID})
			}
			printInfo("DRY RUN: Would update cache settings for zone '%s'", zoneID)
			return nil
		}

		if err := svc.UpdateSettings(ctx, opts...); err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to update cache settings: %v", err))
			}
			return fmt.Errorf("failed to update cache settings: %w", err)
		}

		if JSONOutput {
			return printSuccessJSON("Cache settings updated", map[string]string{"zone_id": zoneID})
		}
		printSuccess("Cache settings updated for zone '%s'", zoneID)
		return nil
	}

	// No update flags — display current settings
	settings, err := svc.GetSettings(ctx)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get cache settings: %v", err))
		}
		return fmt.Errorf("failed to get cache settings: %w", err)
	}

	if JSONOutput {
		return printJSON(settings)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SETTING\tVALUE")
	fmt.Fprintf(w, "Browser Cache TTL\t%d seconds\n", settings.BrowserCacheTTL)
	devModeStr := "off"
	if settings.DevelopmentMode > 0 {
		devModeStr = "on"
	}
	fmt.Fprintf(w, "Development Mode\t%s\n", devModeStr)
	fmt.Fprintf(w, "Cache Level\t%s\n", settings.CacheLevel)
	fmt.Fprintf(w, "Minify CSS\t%v\n", settings.MinifyCss)
	fmt.Fprintf(w, "Minify JS\t%v\n", settings.MinifyJs)
	fmt.Fprintf(w, "Minify HTML\t%v\n", settings.MinifyHtml)
	w.Flush()

	return nil
}
