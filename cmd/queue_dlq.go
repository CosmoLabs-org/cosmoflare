package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	queueConsumerSettingsFile string
	queueDLQConsumerName      string
	queueDLQProducerName      string
	queueDLQClear             bool
)

var queueConsumerCmd = &cobra.Command{
	Use:   "consumer",
	Short: "Manage queue consumers",
	Long: `Manage consumers (Workers) subscribed to a queue.

Commands:
  update  Update consumer settings
  remove  Remove a consumer from a queue

Examples:
  cosmoflare queue consumer update my-queue my-consumer --settings-file settings.json
  cosmoflare queue consumer remove my-queue my-consumer`,
}

var queueConsumerUpdateCmd = &cobra.Command{
	Use:   "update [queue-name] [consumer-name]",
	Short: "Update consumer settings",
	Long: `Update the settings of an existing queue consumer.

The settings are read from a JSON file provided with --settings-file. The
file must contain an object with the following fields:

  batch_size         (optional) number of messages per batch
  max_retries         (optional) maximum delivery retries
  max_wait_time_ms    (optional) maximum time to wait before delivering a batch

Example settings.json:
  {"batch_size": 10, "max_retries": 3, "max_wait_time_ms": 5000}

Examples:
  cosmoflare queue consumer update my-queue my-consumer --settings-file settings.json
  cosmoflare queue consumer update my-queue my-consumer --settings-file settings.json --json`,
	Args: cobra.ExactArgs(2),
	RunE: runQueueConsumerUpdate,
}

var queueConsumerRemoveCmd = &cobra.Command{
	Use:   "remove [queue-name] [consumer-name]",
	Short: "Remove a consumer from a queue",
	Long: `Remove a consumer (Worker) from a queue.

Examples:
  cosmoflare queue consumer remove my-queue my-consumer
  cosmoflare queue consumer remove my-queue my-consumer --json`,
	Args: cobra.ExactArgs(2),
	RunE: runQueueConsumerRemove,
}

var queueDlqCmd = &cobra.Command{
	Use:   "dlq [queue-name]",
	Short: "Show or configure dead letter queue settings",
	Long: `Show or configure the dead letter queue (DLQ) bindings for a queue.

With no flags, prints the queue's current consumer DLQ bindings (read via
'queue get'). With --consumer-dlq and/or --producer-dlq, configures the
named DLQ bindings. With --clear, removes both DLQ bindings. --clear is
mutually exclusive with --consumer-dlq and --producer-dlq.

Examples:
  cosmoflare queue dlq my-queue
  cosmoflare queue dlq my-queue --consumer-dlq my-dlq
  cosmoflare queue dlq my-queue --producer-dlq my-dlq --json
  cosmoflare queue dlq my-queue --clear`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueDlq,
}

func init() {
	queueCmd.AddCommand(queueConsumerCmd)
	queueConsumerCmd.AddCommand(queueConsumerUpdateCmd)
	queueConsumerCmd.AddCommand(queueConsumerRemoveCmd)
	queueCmd.AddCommand(queueDlqCmd)

	queueConsumerUpdateCmd.Flags().StringVar(&queueConsumerSettingsFile, "settings-file", "", "JSON file containing consumer settings")
	_ = queueConsumerUpdateCmd.MarkFlagRequired("settings-file")

	queueDlqCmd.Flags().StringVar(&queueDLQConsumerName, "consumer-dlq", "", "Name of the dead letter queue for consumer delivery failures")
	queueDlqCmd.Flags().StringVar(&queueDLQProducerName, "producer-dlq", "", "Name of the dead letter queue for producer delivery failures")
	queueDlqCmd.Flags().BoolVar(&queueDLQClear, "clear", false, "Remove DLQ bindings")
}

func runQueueConsumerUpdate(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	consumerName := args[1]

	data, err := os.ReadFile(queueConsumerSettingsFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", queueConsumerSettingsFile, err)
	}

	var settings cosmoflare.QueueConsumerSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse %s as consumer settings: %w", queueConsumerSettingsFile, err)
	}

	svc, err := getQueueService()
	if err != nil {
		return fmt.Errorf("failed to create Queue service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would update consumer", map[string]interface{}{
				"queue":    queueName,
				"consumer": consumerName,
				"settings": settings,
			})
		}
		printInfo("DRY RUN: Would update consumer '%s' on queue '%s'", consumerName, queueName)
		return nil
	}

	c, err := svc.UpdateConsumer(context.Background(), queueName, consumerName, settings)
	if err != nil {
		return outErr("failed to update consumer", err)
	}

	if JSONOutput {
		return printSuccessJSON("Consumer updated successfully", c)
	}
	printSuccess("Consumer '%s' updated on queue '%s'", consumerName, queueName)
	return nil
}

func runQueueConsumerRemove(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	consumerName := args[1]

	svc, err := getQueueService()
	if err != nil {
		return fmt.Errorf("failed to create Queue service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would remove consumer", map[string]string{
				"queue":    queueName,
				"consumer": consumerName,
			})
		}
		printInfo("DRY RUN: Would remove consumer '%s' from queue '%s'", consumerName, queueName)
		return nil
	}

	if err := svc.DeleteConsumer(context.Background(), queueName, consumerName); err != nil {
		return outErr("failed to remove consumer", err)
	}

	if JSONOutput {
		return printSuccessJSON("Consumer removed successfully", map[string]string{
			"queue":    queueName,
			"consumer": consumerName,
		})
	}
	printSuccess("Consumer '%s' removed from queue '%s'", consumerName, queueName)
	return nil
}

func runQueueDlq(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	if queueDLQClear && (queueDLQConsumerName != "" || queueDLQProducerName != "") {
		return fmt.Errorf("--clear cannot be combined with --consumer-dlq or --producer-dlq")
	}

	svc, err := getQueueService()
	if err != nil {
		return fmt.Errorf("failed to create Queue service: %w", err)
	}

	if !queueDLQClear && queueDLQConsumerName == "" && queueDLQProducerName == "" {
		q, err := svc.Get(context.Background(), queueName)
		if err != nil {
			return outErr("failed to get queue", err)
		}

		if JSONOutput {
			return printJSON(q)
		}
		printInfo("Queue '%s' DLQ configuration:", q.Name)
		if len(q.Consumers) == 0 {
			printInfo("  no consumers configured")
		}
		for _, c := range q.Consumers {
			fmt.Printf("  consumer %s dead_letter_queue: %s\n", c.Name, c.DeadLetterQueue)
		}
		return nil
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would configure DLQ", map[string]interface{}{
				"queue":        queueName,
				"consumer_dlq": queueDLQConsumerName,
				"producer_dlq": queueDLQProducerName,
				"clear":        queueDLQClear,
			})
		}
		if queueDLQClear {
			printInfo("DRY RUN: Would clear DLQ bindings for queue '%s'", queueName)
		} else {
			printInfo("DRY RUN: Would configure DLQ for queue '%s'", queueName)
		}
		return nil
	}

	q, err := svc.ConfigureDLQ(context.Background(), queueName, queueDLQConsumerName, queueDLQProducerName, queueDLQClear)
	if err != nil {
		return outErr("failed to configure DLQ", err)
	}

	if JSONOutput {
		return printSuccessJSON("DLQ configured successfully", q)
	}
	if queueDLQClear {
		printSuccess("DLQ bindings cleared for queue '%s'", queueName)
	} else {
		printSuccess("DLQ configured for queue '%s'", queueName)
	}
	return nil
}
