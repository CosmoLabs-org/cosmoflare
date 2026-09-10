package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// queueSendMaxMessageBytes mirrors the Cloudflare Queues per-message size
// cap (128 KB base-10, i.e. 128,000 bytes) enforced by the API. See
// pkg/cosmoflare queue.go maxQueueMessageBytes.
const queueSendMaxMessageBytes = 128_000

// queueSendMetadataBytes approximates the internal Queues metadata
// overhead (~100 bytes) counted against the per-message size cap.
const queueSendMetadataBytes = 100

var (
	queueSendBody         string
	queueSendFile         string
	queueSendContentType  string
	queueSendDelaySeconds int
)

var queueSendCmd = &cobra.Command{
	Use:   "send [queue-name]",
	Short: "Send a message to a queue",
	Long: `Send a single message to a Cloudflare Queue.

The message body is provided with --body inline or read from a file with
--file. These two flags are mutually exclusive: setting both is an error.

An optional --content-type (e.g. application/json) and --delay-seconds
(deferred delivery, up to 43200) may be set.

Messages larger than 128,000 bytes (base-10, including ~100 bytes of
internal Queues metadata) exceed the Cloudflare Queues limit; a warning
is printed but the message is still sent and the API makes the final call.

Examples:
  cosmoflare queue send my-queue --body 'hello world'
  cosmoflare queue send my-queue --body 'hello' --content-type text/plain
  cosmoflare queue send my-queue --body 'hello' --delay-seconds 60
  cosmoflare queue send my-queue --file message.json --content-type application/json
  cosmoflare queue send my-queue --file message.json --json`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueSend,
}

var queueSendBatchCmd = &cobra.Command{
	Use:   "send-batch [queue-name]",
	Short: "Send a batch of messages to a queue",
	Long: `Send up to 100 messages to a Cloudflare Queue in one call.

The messages are read from a JSON file provided with --file. The file
must contain a JSON array of objects with the following fields:

  body           (required)  the message body as a string
  content_type   (optional)  e.g. "application/json"
  delay_seconds  (optional)  defer delivery by up to 43200 seconds

Example messages.json:
  [
    {"body": "first message"},
    {"body": "{\"event\":\"signup\"}", "content_type": "application/json"},
    {"body": "delayed", "delay_seconds": 300}
  ]

The batch is limited to 100 messages per call (the Cloudflare Queues API
cap); larger batches must be split into multiple send-batch calls.

Example:
  cosmoflare queue send-batch my-queue --file messages.json
  cosmoflare queue send-batch my-queue --file messages.json --json`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueSendBatch,
}

func init() {
	queueCmd.AddCommand(queueSendCmd)
	queueCmd.AddCommand(queueSendBatchCmd)

	queueSendCmd.Flags().StringVar(&queueSendBody, "body", "", "Message body (inline string)")
	queueSendCmd.Flags().StringVar(&queueSendFile, "file", "", "Read the message body from this file")
	queueSendCmd.Flags().StringVar(&queueSendContentType, "content-type", "", "Content type of the message (e.g. application/json)")
	queueSendCmd.Flags().IntVar(&queueSendDelaySeconds, "delay-seconds", 0, "Delay delivery by this many seconds (max 43200)")

	queueSendBatchCmd.Flags().StringVar(&queueSendFile, "file", "", "JSON file containing an array of messages to send")
	_ = queueSendBatchCmd.MarkFlagRequired("file")
}

// warnQueueSendSize prints a soft warning when a message body would
// exceed the Cloudflare Queues per-message size cap. The message is
// still sent; the API makes the final call.
func warnQueueSendSize(bodyLen int) {
	if bodyLen+queueSendMetadataBytes > queueSendMaxMessageBytes {
		printWarning("message size %d bytes (body %d + ~%d bytes internal Queues metadata) exceeds the 128,000-byte (128 KB, base-10) Cloudflare Queues limit; the API may reject it",
			bodyLen+queueSendMetadataBytes, bodyLen, queueSendMetadataBytes)
	}
}

func runQueueSend(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	if queueSendBody != "" && queueSendFile != "" {
		return fmt.Errorf("--body and --file are mutually exclusive: provide the message inline with --body or from a file with --file, not both")
	}
	if queueSendBody == "" && queueSendFile == "" {
		return fmt.Errorf("one of --body or --file is required")
	}

	body := queueSendBody
	if queueSendFile != "" {
		data, err := os.ReadFile(queueSendFile)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", queueSendFile, err)
		}
		body = string(data)
	}

	warnQueueSendSize(len(body))

	svc, err := getQueueService()
	if err != nil {
		return fmt.Errorf("failed to create Queue service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would send message to queue", map[string]interface{}{
				"queue":         queueName,
				"body_size":     len(body),
				"content_type":  queueSendContentType,
				"delay_seconds": queueSendDelaySeconds,
			})
		}
		printInfo("DRY RUN: Would send %d-byte message to queue '%s'", len(body), queueName)
		return nil
	}

	res, err := svc.Send(context.Background(), queueName, cosmoflare.QueueMessage{
		Body:         body,
		ContentType:  queueSendContentType,
		DelaySeconds: queueSendDelaySeconds,
	})
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to send message: %v", err))
		}
		return fmt.Errorf("failed to send message: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Message sent successfully", res)
	}
	if res.MessageID != "" {
		printSuccess("Message sent to queue '%s' (message ID: %s)", queueName, res.MessageID)
	} else {
		printSuccess("Message sent to queue '%s'", queueName)
	}
	return nil
}

func runQueueSendBatch(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	if queueSendFile == "" {
		return fmt.Errorf("--file is required")
	}

	data, err := os.ReadFile(queueSendFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", queueSendFile, err)
	}

	var msgs []cosmoflare.QueueMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return fmt.Errorf("failed to parse %s as a JSON array of messages: %w", queueSendFile, err)
	}
	if len(msgs) == 0 {
		return fmt.Errorf("no messages found in %s: expected a JSON array with at least one message", queueSendFile)
	}

	for _, m := range msgs {
		warnQueueSendSize(len(m.Body))
	}

	svc, err := getQueueService()
	if err != nil {
		return fmt.Errorf("failed to create Queue service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would send batch to queue", map[string]interface{}{
				"queue":    queueName,
				"messages": len(msgs),
			})
		}
		printInfo("DRY RUN: Would send batch of %d message(s) to queue '%s'", len(msgs), queueName)
		return nil
	}

	res, err := svc.SendBatch(context.Background(), queueName, msgs)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to send batch: %v", err))
		}
		return fmt.Errorf("failed to send batch: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Batch sent successfully", res)
	}
	printSuccess("Sent %d message(s) to queue '%s'", res.Sent, queueName)
	return nil
}
