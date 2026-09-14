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

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage Cloudflare Queues",
	Long: `Queue management for Cloudflare's message queues.

Commands:
  create     Create a queue
  list       List queues
  get        Get queue details
  update     Rename a queue
  delete     Delete a queue
  consumers  List consumers for a queue
  consumer   Manage queue consumers (update, remove)
  dlq        Show or configure dead letter queue settings
  send       Send a message to a queue
  send-batch Send up to 100 messages to a queue

Examples:
  cosmoflare queue create my-queue
  cosmoflare queue list --json
  cosmoflare queue get my-queue
  cosmoflare queue update my-queue --name new-name
  cosmoflare queue delete my-queue --force
  cosmoflare queue consumers my-queue
  cosmoflare queue consumer update my-queue my-consumer --settings-file settings.json
  cosmoflare queue consumer remove my-queue my-consumer
  cosmoflare queue dlq my-queue --consumer-dlq my-dlq
  cosmoflare queue send my-queue --body 'hello'
  cosmoflare queue send-batch my-queue --file messages.json`,
}

var (
	queueForce   bool
	queueNewName string
)

var queueCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a queue",
	Long: `Create a new Cloudflare Queue.

The name must be unique within your account.

Examples:
  cosmoflare queue create my-queue
  cosmoflare queue create production-events --json`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueCreate,
}

var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all queues",
	Long: `List all queues in the current account.

Examples:
  cosmoflare queue list
  cosmoflare queue list --json`,
	RunE: runQueueList,
}

var queueGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get queue details",
	Long: `Get details of a queue by name.

Examples:
  cosmoflare queue get my-queue
  cosmoflare queue get my-queue --json`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueGet,
}

var queueUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Rename a queue",
	Long: `Rename an existing queue.

Examples:
  cosmoflare queue update old-name --name new-name
  cosmoflare queue update old-name --name new-name --json`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueUpdate,
}

var queueDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a queue",
	Long: `Delete a queue and all its messages.

WARNING: This action is irreversible. All messages will be lost.

Examples:
  cosmoflare queue delete my-queue
  cosmoflare queue delete my-queue --force`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueDelete,
}

var queueConsumersCmd = &cobra.Command{
	Use:   "consumers [queue-name]",
	Short: "List consumers for a queue",
	Long: `List all consumers (Workers) subscribed to a queue.

Examples:
  cosmoflare queue consumers my-queue
  cosmoflare queue consumers my-queue --json`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueConsumers,
}

func init() {
	rootCmd.AddCommand(queueCmd)

	queueCmd.AddCommand(queueCreateCmd)
	queueCmd.AddCommand(queueListCmd)
	queueCmd.AddCommand(queueGetCmd)
	queueCmd.AddCommand(queueUpdateCmd)
	queueCmd.AddCommand(queueDeleteCmd)
	queueCmd.AddCommand(queueConsumersCmd)

	queueDeleteCmd.Flags().BoolVar(&queueForce, "force", false, "Skip confirmation prompt")
	queueUpdateCmd.Flags().StringVar(&queueNewName, "name", "", "New name for the queue")
	_ = queueUpdateCmd.MarkFlagRequired("name")
}

func getQueueService() (*cosmoflare.QueueService, error) {
	return cosmoflare.NewQueueServiceFromCreds(AccountID, APIToken)
}

func runQueueCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getQueueService()
	if err != nil {
		return outErr("failed to create Queue service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create queue", func() any {
			return map[string]string{"name": name}
		}, func() {
			printInfo("DRY RUN: Would create queue '%s'", name)
		})
	}

	q, err := svc.Create(context.Background(), name)
	if err != nil {
		return outErr("failed to create queue", err)
	}

	return outPayload("Queue created successfully", func() any {
		return q
	}, func() {
		printSuccess("Queue '%s' created (ID: %s)", q.Name, q.ID)
	})
}

func runQueueList(cmd *cobra.Command, args []string) error {
	svc, err := getQueueService()
	if err != nil {
		return outErr("failed to create Queue service", err)
	}

	queues, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list queues", err)
	}

	return outResult(queues, func() {
		if len(queues) == 0 {
			printInfo("No queues found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPRODUCERS\tCONSUMERS")
		for _, q := range queues {
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\n", q.ID, q.Name, q.ProducersTotalCount, q.ConsumersTotalCount)
		}
		w.Flush()
		printInfo("Total: %d queue(s)", len(queues))
	})
}

func runQueueGet(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	svc, err := getQueueService()
	if err != nil {
		return outErr("failed to create Queue service", err)
	}

	q, err := svc.Get(context.Background(), queueName)
	if err != nil {
		return outErr("failed to get queue", err)
	}

	return outResult(q, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", q.ID)
		fmt.Fprintf(w, "Name:\t%s\n", q.Name)
		fmt.Fprintf(w, "Producers:\t%d\n", q.ProducersTotalCount)
		fmt.Fprintf(w, "Consumers:\t%d\n", q.ConsumersTotalCount)
		if q.CreatedOn != nil {
			fmt.Fprintf(w, "Created:\t%s\n", q.CreatedOn.Format("2006-01-02 15:04:05 UTC"))
		}
		if q.ModifiedOn != nil {
			fmt.Fprintf(w, "Modified:\t%s\n", q.ModifiedOn.Format("2006-01-02 15:04:05 UTC"))
		}
		w.Flush()
	})
}

func runQueueUpdate(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	if queueNewName == "" {
		return fmt.Errorf("--name flag is required")
	}

	svc, err := getQueueService()
	if err != nil {
		return outErr("failed to create Queue service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would rename queue", func() any {
			return map[string]string{
				"old_name": queueName,
				"new_name": queueNewName,
			}
		}, func() {
			printInfo("DRY RUN: Would rename queue '%s' to '%s'", queueName, queueNewName)
		})
	}

	q, err := svc.Update(context.Background(), queueName, queueNewName)
	if err != nil {
		return outErr("failed to update queue", err)
	}

	return outPayload("Queue updated successfully", func() any {
		return q
	}, func() {
		printSuccess("Queue renamed to '%s' (ID: %s)", q.Name, q.ID)
	})
}

func runQueueDelete(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	if !queueForce && !DryRun {
		fmt.Printf("Are you sure you want to delete queue '%s' and all its messages? [y/N]: ", queueName)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Queue deletion cancelled")
			return nil
		}
	}

	svc, err := getQueueService()
	if err != nil {
		return outErr("failed to create Queue service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete queue", func() any {
			return map[string]string{"name": queueName}
		}, func() {
			printInfo("DRY RUN: Would delete queue '%s'", queueName)
		})
	}

	if err := svc.Delete(context.Background(), queueName); err != nil {
		return outErr("failed to delete queue", err)
	}

	return outPayload("Queue deleted successfully", func() any {
		return map[string]string{"name": queueName}
	}, func() {
		printSuccess("Queue '%s' deleted successfully!", queueName)
	})
}

func runQueueConsumers(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	svc, err := getQueueService()
	if err != nil {
		return outErr("failed to create Queue service", err)
	}

	consumers, err := svc.ListConsumers(context.Background(), queueName)
	if err != nil {
		return outErr("failed to list consumers", err)
	}

	return outResult(consumers, func() {
		if len(consumers) == 0 {
			printInfo("No consumers found for queue '%s'", queueName)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSCRIPT\tENVIRONMENT\tBATCH_SIZE\tMAX_RETRIES\tDEAD_LETTER")
		for _, c := range consumers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%s\n",
				c.Name, c.ScriptName, c.Environment,
				c.Settings.BatchSize, c.Settings.MaxRetries, c.DeadLetterQueue)
		}
		w.Flush()
		printInfo("Total: %d consumer(s)", len(consumers))
	})
}
