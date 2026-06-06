package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// ---------------------------------------------------------------------------
// Parent command: cosmoflare ai
// ---------------------------------------------------------------------------

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Manage Workers AI models and AI Gateway",
	Long: `Run AI models at the edge with Workers AI and manage AI Gateway configurations.

Subcommand groups:
  models    Browse and inspect available AI models
  run       Run inference on a Workers AI model
  gateway   Manage AI Gateway configurations and logs

Examples:
  cosmoflare ai models list
  cosmoflare ai models list --filter text-generation
  cosmoflare ai models get @cf/meta/llama-3-8b-instruct
  cosmoflare ai run @cf/meta/llama-3-8b-instruct --prompt "Hello, world!"
  cosmoflare ai gateway list
  cosmoflare ai gateway create my-gateway --cache-ttl 300
  cosmoflare ai gateway logs my-gateway --limit 50 --json`,
}

// ---------------------------------------------------------------------------
// ai models
// ---------------------------------------------------------------------------

var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Browse available Workers AI models",
	Long: `List and inspect available Workers AI models.

Examples:
  cosmoflare ai models list
  cosmoflare ai models list --filter text-generation --json
  cosmoflare ai models get @cf/meta/llama-3-8b-instruct`,
}

var (
	aiModelsFilter string
)

var aiModelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available AI models",
	Long: `List all available Workers AI models, optionally filtered by task type.

Common task types:
  text-generation, text-classification, text-embedding,
  image-classification, image-to-text, translation,
  speech-recognition, summarization, object-detection

Examples:
  cosmoflare ai models list
  cosmoflare ai models list --filter text-generation
  cosmoflare ai models list --filter image-classification --json`,
	RunE: runAIModelsList,
}

var aiModelsGetCmd = &cobra.Command{
	Use:   "get [model-name]",
	Short: "Get details for a specific AI model",
	Long: `Retrieve detailed information about a specific Workers AI model.

The model name is the full qualified name (e.g. @cf/meta/llama-3-8b-instruct).

Examples:
  cosmoflare ai models get @cf/meta/llama-3-8b-instruct
  cosmoflare ai models get @cf/openai/whisper --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAIModelsGet,
}

// ---------------------------------------------------------------------------
// ai run
// ---------------------------------------------------------------------------

var (
	aiRunPrompt string
	aiRunSystem string
)

var aiRunCmd = &cobra.Command{
	Use:   "run [model-name]",
	Short: "Run inference on a Workers AI model",
	Long: `Run inference on a Workers AI model with a text prompt.

Supports single-prompt and multi-turn chat (--system + --prompt) modes.
For binary output models (image generation), the raw response is written to stdout.

Examples:
  cosmoflare ai run @cf/meta/llama-3-8b-instruct --prompt "What is Go?"
  cosmoflare ai run @cf/meta/llama-3-8b-instruct --system "You are a pirate" --prompt "Tell me about Go"
  cosmoflare ai run @cf/meta/llama-3-8b-instruct --prompt "Hello" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAIRun,
}

// ---------------------------------------------------------------------------
// ai gateway
// ---------------------------------------------------------------------------

var aiGatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage AI Gateway configurations",
	Long: `Create, list, and manage AI Gateway configurations for caching, rate limiting,
and logging AI API requests.

Examples:
  cosmoflare ai gateway list
  cosmoflare ai gateway create my-gateway --cache-ttl 300
  cosmoflare ai gateway delete my-gateway --force
  cosmoflare ai gateway logs my-gateway --limit 50`,
}

var aiGatewayListCmd = &cobra.Command{
	Use:   "list",
	Short: "List AI Gateway configurations",
	Long: `List all AI Gateway configurations in the current account.

Examples:
  cosmoflare ai gateway list
  cosmoflare ai gateway list --json`,
	RunE: runAIGatewayList,
}

var (
	aiGatewayCacheTTL    int
	aiGatewayRateLimit   int
	aiGatewayRateWindow  int
	aiGatewayCollectLogs bool
	aiGatewayForce       bool
	aiGatewayLogsLimit   int
)

var aiGatewayCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create an AI Gateway",
	Long: `Create a new AI Gateway configuration with optional caching and rate limiting.

Examples:
  cosmoflare ai gateway create my-gateway
  cosmoflare ai gateway create prod-gw --cache-ttl 300 --rate-limit 100 --rate-window 60
  cosmoflare ai gateway create my-gateway --collect-logs --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAIGatewayCreate,
}

var aiGatewayDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete an AI Gateway",
	Long: `Delete an AI Gateway configuration.

Examples:
  cosmoflare ai gateway delete my-gateway
  cosmoflare ai gateway delete my-gateway --force`,
	Args: cobra.ExactArgs(1),
	RunE: runAIGatewayDelete,
}

var aiGatewayLogsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "View AI Gateway logs",
	Long: `View recent log entries for an AI Gateway, showing model usage,
latency, cache hits, and costs.

Examples:
  cosmoflare ai gateway logs my-gateway
  cosmoflare ai gateway logs my-gateway --limit 100
  cosmoflare ai gateway logs my-gateway --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAIGatewayLogs,
}

// ---------------------------------------------------------------------------
// init — registration
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(aiCmd)

	// ai models
	aiCmd.AddCommand(aiModelsCmd)
	aiModelsCmd.AddCommand(aiModelsListCmd)
	aiModelsCmd.AddCommand(aiModelsGetCmd)
	aiModelsListCmd.Flags().StringVar(&aiModelsFilter, "filter", "", "Filter models by task type (e.g. text-generation, image-classification)")

	// ai run
	aiCmd.AddCommand(aiRunCmd)
	aiRunCmd.Flags().StringVar(&aiRunPrompt, "prompt", "", "Text prompt for inference")
	aiRunCmd.Flags().StringVar(&aiRunSystem, "system", "", "System message for chat models")

	// ai gateway
	aiCmd.AddCommand(aiGatewayCmd)
	aiGatewayCmd.AddCommand(aiGatewayListCmd)
	aiGatewayCmd.AddCommand(aiGatewayCreateCmd)
	aiGatewayCmd.AddCommand(aiGatewayDeleteCmd)
	aiGatewayCmd.AddCommand(aiGatewayLogsCmd)

	aiGatewayCreateCmd.Flags().IntVar(&aiGatewayCacheTTL, "cache-ttl", 0, "Cache TTL in seconds (0 = no caching)")
	aiGatewayCreateCmd.Flags().IntVar(&aiGatewayRateLimit, "rate-limit", 0, "Rate limit (requests per window)")
	aiGatewayCreateCmd.Flags().IntVar(&aiGatewayRateWindow, "rate-window", 60, "Rate limit window in seconds")
	aiGatewayCreateCmd.Flags().BoolVar(&aiGatewayCollectLogs, "collect-logs", false, "Enable log collection")

	aiGatewayDeleteCmd.Flags().BoolVar(&aiGatewayForce, "force", false, "Skip confirmation prompt")

	aiGatewayLogsCmd.Flags().IntVar(&aiGatewayLogsLimit, "limit", 25, "Number of log entries to retrieve")
}

// ---------------------------------------------------------------------------
// Service factory
// ---------------------------------------------------------------------------

func getAIService() (*cosmoflare.AIService, error) {
	return cosmoflare.NewAIServiceFromCreds(AccountID, APIToken)
}

// ---------------------------------------------------------------------------
// Run functions
// ---------------------------------------------------------------------------

func runAIModelsList(cmd *cobra.Command, args []string) error {
	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	models, err := svc.ListModels(context.Background(), aiModelsFilter)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	if JSONOutput {
		return printJSON(models)
	}

	if len(models) == 0 {
		printInfo("No AI models found")
		if aiModelsFilter != "" {
			printInfo("Try a different --filter value or omit it to list all models")
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MODEL\tTASK\tDESCRIPTION")
	for _, m := range models {
		desc := m.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, m.Task.Name, desc)
	}
	w.Flush()
	return nil
}

func runAIModelsGet(cmd *cobra.Command, args []string) error {
	modelName := args[0]

	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	model, err := svc.GetModel(context.Background(), modelName)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	if JSONOutput {
		return printJSON(model)
	}

	fmt.Printf("Name:        %s\n", model.Name)
	fmt.Printf("Task:        %s\n", model.Task.Name)
	if model.Description != "" {
		fmt.Printf("Description: %s\n", model.Description)
	}
	if model.Task.Description != "" {
		fmt.Printf("Task Info:   %s\n", model.Task.Description)
	}
	if len(model.Properties) > 0 {
		fmt.Println("Properties:")
		for _, p := range model.Properties {
			fmt.Printf("  %s: %s\n", p.PropertyID, p.Value)
		}
	}
	return nil
}

func runAIRun(cmd *cobra.Command, args []string) error {
	modelName := args[0]

	if aiRunPrompt == "" {
		return fmt.Errorf("--prompt is required\n\nUsage: cosmoflare ai run %s --prompt \"Your prompt here\"", modelName)
	}

	if DryRun {
		if JSONOutput {
			return printJSON(map[string]interface{}{
				"dry_run": true,
				"action":  "run_inference",
				"model":   modelName,
				"prompt":  aiRunPrompt,
				"system":  aiRunSystem,
			})
		}
		printInfo("DRY RUN: Would run inference on model '%s'", modelName)
		printInfo("  Prompt: %s", aiRunPrompt)
		if aiRunSystem != "" {
			printInfo("  System: %s", aiRunSystem)
		}
		return nil
	}

	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	input := &cosmoflare.AIInferenceInput{}
	if aiRunSystem != "" {
		// Multi-turn chat mode with system message
		input.Messages = []cosmoflare.AIMessage{
			{Role: "system", Content: aiRunSystem},
			{Role: "user", Content: aiRunPrompt},
		}
	} else {
		input.Prompt = aiRunPrompt
	}

	result, err := svc.RunInference(context.Background(), modelName, input)
	if err != nil {
		return fmt.Errorf("failed to run inference: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{
			"success":  true,
			"model":    modelName,
			"response": result.Response,
		})
	}

	if result.Response != "" {
		fmt.Println(result.Response)
	} else {
		// For non-text responses, output the raw result
		fmt.Printf("%s\n", result.Result)
	}
	return nil
}

func runAIGatewayList(cmd *cobra.Command, args []string) error {
	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	gateways, err := svc.ListGateways(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list gateways: %w", err)
	}

	if JSONOutput {
		return printJSON(gateways)
	}

	if len(gateways) == 0 {
		printInfo("No AI gateways found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCACHE TTL\tCREATED")
	for _, gw := range gateways {
		cacheTTL := "disabled"
		if gw.CacheTTL > 0 {
			cacheTTL = fmt.Sprintf("%ds", gw.CacheTTL)
		}
		created := gw.CreatedAt
		if len(created) > 10 {
			created = created[:10]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", gw.ID, gw.Name, cacheTTL, created)
	}
	w.Flush()
	return nil
}

func runAIGatewayCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		if JSONOutput {
			return printJSON(map[string]interface{}{
				"dry_run":      true,
				"action":       "create_gateway",
				"name":         name,
				"cache_ttl":    aiGatewayCacheTTL,
				"rate_limit":   aiGatewayRateLimit,
				"rate_window":  aiGatewayRateWindow,
				"collect_logs": aiGatewayCollectLogs,
			})
		}
		printInfo("DRY RUN: Would create AI gateway '%s'", name)
		if aiGatewayCacheTTL > 0 {
			printInfo("  Cache TTL: %ds", aiGatewayCacheTTL)
		}
		if aiGatewayRateLimit > 0 {
			printInfo("  Rate limit: %d requests per %ds", aiGatewayRateLimit, aiGatewayRateWindow)
		}
		return nil
	}

	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	params := cosmoflare.AIGatewayCreateParams{
		CacheTTL:           aiGatewayCacheTTL,
		RateLimitLimit:     aiGatewayRateLimit,
		RateLimitInterval:  aiGatewayRateWindow,
	}
	if aiGatewayCollectLogs {
		params.CollectLogs = &aiGatewayCollectLogs
	}

	gw, err := svc.CreateGateway(context.Background(), name, params)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{"success": true, "data": gw})
	}

	printSuccess("Created AI gateway '%s'", gw.ID)
	if gw.CacheTTL > 0 {
		printInfo("Cache TTL: %ds", gw.CacheTTL)
	}
	return nil
}

func runAIGatewayDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		if JSONOutput {
			return printJSON(map[string]interface{}{"dry_run": true, "action": "delete_gateway", "name": name})
		}
		printInfo("DRY RUN: Would delete AI gateway '%s'", name)
		return nil
	}

	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	if err := svc.DeleteGateway(context.Background(), name); err != nil {
		return fmt.Errorf("failed to delete gateway: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{"success": true, "message": fmt.Sprintf("Gateway '%s' deleted", name)})
	}
	printSuccess("Deleted AI gateway '%s'", name)
	return nil
}

func runAIGatewayLogs(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getAIService()
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	logs, err := svc.GetGatewayLogs(context.Background(), name, aiGatewayLogsLimit)
	if err != nil {
		return fmt.Errorf("failed to get gateway logs: %w", err)
	}

	if JSONOutput {
		return printJSON(logs)
	}

	if len(logs) == 0 {
		printInfo("No logs found for gateway '%s'", name)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tMODEL\tSTATUS\tCACHED\tTOKENS\tCOST")
	for _, l := range logs {
		model := l.Model
		if len(model) > 40 {
			// Shorten long model names for display
			parts := strings.Split(model, "/")
			if len(parts) >= 3 {
				model = parts[len(parts)-1]
			}
		}
		cached := "no"
		if l.Cached {
			cached = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t$%.4f\n",
			l.ID, model, l.StatusCode, cached, l.Tokens, l.Cost)
	}
	w.Flush()
	return nil
}
