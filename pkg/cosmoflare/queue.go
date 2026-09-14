package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Queue represents a Cloudflare Queue.
type Queue struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	CreatedOn           *time.Time      `json:"created_on"`
	ModifiedOn          *time.Time      `json:"modified_on"`
	ProducersTotalCount int             `json:"producers_total_count"`
	Producers           []QueueProducer `json:"producers"`
	ConsumersTotalCount int             `json:"consumers_total_count"`
	Consumers           []QueueConsumer `json:"consumers"`
}

// QueueProducer represents a producer bound to a queue.
type QueueProducer struct {
	Service     string `json:"service"`
	Environment string `json:"environment"`
}

// QueueConsumer represents a consumer (Worker) subscribed to a queue.
type QueueConsumer struct {
	Name            string                `json:"name"`
	Service         string                `json:"service"`
	ScriptName      string                `json:"script_name"`
	Environment     string                `json:"environment"`
	QueueName       string                `json:"queue_name"`
	Settings        QueueConsumerSettings `json:"settings"`
	CreatedOn       *time.Time            `json:"created_on"`
	DeadLetterQueue string                `json:"dead_letter_queue"`
}

// QueueConsumerSettings holds configuration for a queue consumer.
type QueueConsumerSettings struct {
	BatchSize   int `json:"batch_size"`
	MaxRetries  int `json:"max_retries"`
	MaxWaitTime int `json:"max_wait_time_ms"`
}

// QueueService implements Cloudflare Queues operations.
type QueueService struct {
	cf        *cloudflare.API
	accountID string
}

// NewQueueService creates a new Queues service client.
func NewQueueService(api *cloudflare.API, accountID string) (*QueueService, error) {
	if api == nil {
		return nil, validationError("NewQueueService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewQueueService", "account ID is required")
	}
	return &QueueService{cf: api, accountID: accountID}, nil
}

// NewQueueServiceFromCreds creates a QueueService from account ID and API token.
func NewQueueServiceFromCreds(accountID, apiToken string) (*QueueService, error) {
	if accountID == "" {
		return nil, validationError("NewQueueService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewQueueService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewQueueService", "failed to create Cloudflare API client", err)
	}
	return &QueueService{cf: cf, accountID: accountID}, nil
}

// List returns all queues in the account.
func (s *QueueService) List(ctx context.Context) ([]*Queue, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListQueues(ctx, rc, cloudflare.ListQueuesParams{})
	if err != nil {
		return nil, newError("QueueService.List", "failed to list queues", err)
	}

	queues := make([]*Queue, 0, len(results))
	for _, q := range results {
		queues = append(queues, mapQueue(q))
	}
	return queues, nil
}

// Get retrieves a single queue by name.
func (s *QueueService) Get(ctx context.Context, queueName string) (*Queue, error) {
	if queueName == "" {
		return nil, validationError("QueueService.Get", "queue name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetQueue(ctx, rc, queueName)
	if err != nil {
		return nil, newError("QueueService.Get", fmt.Sprintf("failed to get queue %q", queueName), err)
	}

	return mapQueue(result), nil
}

// Create creates a new queue.
func (s *QueueService) Create(ctx context.Context, name string) (*Queue, error) {
	if name == "" {
		return nil, validationError("QueueService.Create", "queue name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.CreateQueue(ctx, rc, cloudflare.CreateQueueParams{Name: name})
	if err != nil {
		return nil, newError("QueueService.Create", fmt.Sprintf("failed to create queue %q", name), err)
	}

	return mapQueue(result), nil
}

// Update renames a queue.
func (s *QueueService) Update(ctx context.Context, queueName, newName string) (*Queue, error) {
	if queueName == "" {
		return nil, validationError("QueueService.Update", "queue name is required")
	}
	if newName == "" {
		return nil, validationError("QueueService.Update", "new queue name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.UpdateQueue(ctx, rc, cloudflare.UpdateQueueParams{
		Name:        queueName,
		UpdatedName: newName,
	})
	if err != nil {
		return nil, newError("QueueService.Update", fmt.Sprintf("failed to update queue %q", queueName), err)
	}

	return mapQueue(result), nil
}

// Delete deletes a queue by name.
func (s *QueueService) Delete(ctx context.Context, queueName string) error {
	if queueName == "" {
		return validationError("QueueService.Delete", "queue name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeleteQueue(ctx, rc, queueName)
	if err != nil {
		return newError("QueueService.Delete", fmt.Sprintf("failed to delete queue %q", queueName), err)
	}
	return nil
}

// ListConsumers returns all consumers for a queue.
func (s *QueueService) ListConsumers(ctx context.Context, queueName string) ([]*QueueConsumer, error) {
	if queueName == "" {
		return nil, validationError("QueueService.ListConsumers", "queue name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListQueueConsumers(ctx, rc, cloudflare.ListQueueConsumersParams{
		QueueName: queueName,
	})
	if err != nil {
		return nil, newError("QueueService.ListConsumers", fmt.Sprintf("failed to list consumers for queue %q", queueName), err)
	}

	consumers := make([]*QueueConsumer, 0, len(results))
	for _, c := range results {
		consumers = append(consumers, mapQueueConsumer(c))
	}
	return consumers, nil
}

// CreateConsumer creates a new consumer for a queue.
func (s *QueueService) CreateConsumer(ctx context.Context, queueName string, consumer QueueConsumer) (*QueueConsumer, error) {
	if queueName == "" {
		return nil, validationError("QueueService.CreateConsumer", "queue name is required")
	}
	if consumer.ScriptName == "" {
		return nil, validationError("QueueService.CreateConsumer", "consumer script name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.CreateQueueConsumer(ctx, rc, cloudflare.CreateQueueConsumerParams{
		QueueName: queueName,
		Consumer: cloudflare.QueueConsumer{
			ScriptName:      consumer.ScriptName,
			Environment:     consumer.Environment,
			Settings:        mapQueueConsumerSettingsToSDK(consumer.Settings),
			DeadLetterQueue: consumer.DeadLetterQueue,
		},
	})
	if err != nil {
		return nil, newError("QueueService.CreateConsumer", fmt.Sprintf("failed to create consumer for queue %q", queueName), err)
	}

	return mapQueueConsumer(result), nil
}

// DeleteConsumer deletes a consumer from a queue.
func (s *QueueService) DeleteConsumer(ctx context.Context, queueName, consumerName string) error {
	if queueName == "" {
		return validationError("QueueService.DeleteConsumer", "queue name is required")
	}
	if consumerName == "" {
		return validationError("QueueService.DeleteConsumer", "consumer name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeleteQueueConsumer(ctx, rc, cloudflare.DeleteQueueConsumerParams{
		QueueName:    queueName,
		ConsumerName: consumerName,
	})
	if err != nil {
		return newError("QueueService.DeleteConsumer", fmt.Sprintf("failed to delete consumer %q from queue %q", consumerName, queueName), err)
	}
	return nil
}

// UpdateConsumer updates the settings of an existing consumer for a queue.
func (s *QueueService) UpdateConsumer(ctx context.Context, queueName, consumerName string, settings QueueConsumerSettings) (*QueueConsumer, error) {
	if queueName == "" {
		return nil, validationError("QueueService.UpdateConsumer", "queue name is required")
	}
	if consumerName == "" {
		return nil, validationError("QueueService.UpdateConsumer", "consumer name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.UpdateQueueConsumer(ctx, rc, cloudflare.UpdateQueueConsumerParams{
		QueueName: queueName,
		Consumer: cloudflare.QueueConsumer{
			Name:     consumerName,
			Settings: mapQueueConsumerSettingsToSDK(settings),
		},
	})
	if err != nil {
		return nil, newError("QueueService.UpdateConsumer", fmt.Sprintf("failed to update consumer %q for queue %q", consumerName, queueName), err)
	}

	return mapQueueConsumer(result), nil
}

// ConfigureDLQ sets or clears the dead letter queue bindings on a queue's
// consumer and/or producer settings. consumerDLQ and producerDLQ are
// optional ("" leaves that binding unchanged); at least one is required
// unless clear is set. clear=true removes both bindings.
func (s *QueueService) ConfigureDLQ(ctx context.Context, queueName, consumerDLQ, producerDLQ string, clear bool) (*Queue, error) {
	if queueName == "" {
		return nil, validationError("QueueService.ConfigureDLQ", "queue name is required")
	}
	if !clear && consumerDLQ == "" && producerDLQ == "" {
		return nil, validationError("QueueService.ConfigureDLQ", "at least one of consumerDLQ or producerDLQ is required unless clear is set")
	}

	queueID, err := s.resolveQueueID(ctx, queueName)
	if err != nil {
		return nil, err
	}

	settings := map[string]interface{}{}
	if clear {
		settings["consumers"] = map[string]string{"dead_letter_queue": ""}
		settings["producers"] = map[string]string{"dead_letter_queue": ""}
	} else {
		if consumerDLQ != "" {
			settings["consumers"] = map[string]string{"dead_letter_queue": consumerDLQ}
		}
		if producerDLQ != "" {
			settings["producers"] = map[string]string{"dead_letter_queue": producerDLQ}
		}
	}

	payload := struct {
		Settings map[string]interface{} `json:"settings"`
	}{Settings: settings}

	uri := fmt.Sprintf("/accounts/%s/queues/%s", s.accountID, queueID)
	raw, err := s.cf.Raw(ctx, http.MethodPut, uri, payload, nil)
	if err != nil {
		return nil, newError("QueueService.ConfigureDLQ", fmt.Sprintf("failed to configure DLQ for queue %q", queueName), err)
	}
	if !raw.Success {
		return nil, newError("QueueService.ConfigureDLQ", fmt.Sprintf("failed to configure DLQ for queue %q", queueName), nil)
	}

	if len(raw.Result) > 0 {
		var sdkQueue cloudflare.Queue
		if err := json.Unmarshal(raw.Result, &sdkQueue); err == nil && sdkQueue.ID != "" {
			return mapQueue(sdkQueue), nil
		}
	}
	return s.Get(ctx, queueName)
}

// mapQueue converts a cloudflare.Queue to our Queue type.
func mapQueue(q cloudflare.Queue) *Queue {
	producers := make([]QueueProducer, 0, len(q.Producers))
	for _, p := range q.Producers {
		producers = append(producers, QueueProducer{
			Service:     p.Service,
			Environment: p.Environment,
		})
	}

	consumers := make([]QueueConsumer, 0, len(q.Consumers))
	for _, c := range q.Consumers {
		consumers = append(consumers, *mapQueueConsumer(c))
	}

	return &Queue{
		ID:                  q.ID,
		Name:                q.Name,
		CreatedOn:           q.CreatedOn,
		ModifiedOn:          q.ModifiedOn,
		ProducersTotalCount: q.ProducersTotalCount,
		Producers:           producers,
		ConsumersTotalCount: q.ConsumersTotalCount,
		Consumers:           consumers,
	}
}

// mapQueueConsumer converts a cloudflare.QueueConsumer to our QueueConsumer type.
func mapQueueConsumer(c cloudflare.QueueConsumer) *QueueConsumer {
	return &QueueConsumer{
		Name:        c.Name,
		Service:     c.Service,
		ScriptName:  c.ScriptName,
		Environment: c.Environment,
		QueueName:   c.QueueName,
		Settings: QueueConsumerSettings{
			BatchSize:   c.Settings.BatchSize,
			MaxRetries:  c.Settings.MaxRetires,
			MaxWaitTime: c.Settings.MaxWaitTime,
		},
		CreatedOn:       c.CreatedOn,
		DeadLetterQueue: c.DeadLetterQueue,
	}
}

// mapQueueConsumerSettingsToSDK converts our settings to the SDK type.
func mapQueueConsumerSettingsToSDK(s QueueConsumerSettings) cloudflare.QueueConsumerSettings {
	return cloudflare.QueueConsumerSettings{
		BatchSize:   s.BatchSize,
		MaxRetires:  s.MaxRetries,
		MaxWaitTime: s.MaxWaitTime,
	}
}

// maxQueueMessageBytes is the documented Cloudflare Queues per-message
// size cap, expressed in base-10 units (128 KB = 128,000 bytes, not
// 128 * 1024). The cap is measured on the wire payload *including*
// ~100 bytes of internal Queues metadata, so an application body must
// stay below this even though the guard below only warns.
// Source: docs/research/2026-09-10-cf-limits-corpus/qwen-results.md
// EDGE CASES item 7.
const maxQueueMessageBytes = 128_000

// queueMessageMetadataBytes approximates the internal Queues metadata
// overhead counted against the per-message size cap.
const queueMessageMetadataBytes = 100

// maxQueueBatchMessages is the hard cap on messages per batch request
// enforced by the Cloudflare Queues messages API.
const maxQueueBatchMessages = 100

// QueueMessage is a message to produce onto a queue.
type QueueMessage struct {
	Body         string `json:"body"`
	ContentType  string `json:"content_type,omitempty"`
	DelaySeconds int    `json:"delay_seconds,omitempty"`
}

// SendMessageResult reports the outcome of a single Send call.
type SendMessageResult struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
}

// SendBatchResult reports the outcome of a SendBatch call.
type SendBatchResult struct {
	Success bool `json:"success"`
	Sent    int  `json:"sent"`
}

// Send produces a single message onto the named queue.
func (s *QueueService) Send(ctx context.Context, queueName string, msg QueueMessage) (*SendMessageResult, error) {
	if queueName == "" {
		return nil, validationError("QueueService.Send", "queue name is required")
	}

	queueID, err := s.resolveQueueID(ctx, queueName)
	if err != nil {
		return nil, err
	}

	warnQueueMessageSize(len(msg.Body))

	uri := fmt.Sprintf("/accounts/%s/queues/%s/messages", s.accountID, queueID)
	raw, err := s.cf.Raw(ctx, http.MethodPost, uri, msg, nil)
	if err != nil {
		return nil, newError("QueueService.Send", fmt.Sprintf("failed to send message to queue %q", queueName), err)
	}
	if !raw.Success {
		return nil, newError("QueueService.Send", fmt.Sprintf("failed to send message to queue %q", queueName), nil)
	}

	result := &SendMessageResult{Success: true}
	if len(raw.Result) > 0 {
		var decoded struct {
			MessageID string `json:"message_id"`
		}
		if json.Unmarshal(raw.Result, &decoded) == nil {
			result.MessageID = decoded.MessageID
		}
	}
	return result, nil
}

// SendBatch produces up to maxQueueBatchMessages messages onto the named
// queue in a single API call.
func (s *QueueService) SendBatch(ctx context.Context, queueName string, msgs []QueueMessage) (*SendBatchResult, error) {
	if queueName == "" {
		return nil, validationError("QueueService.SendBatch", "queue name is required")
	}
	if len(msgs) == 0 {
		return nil, validationError("QueueService.SendBatch", "at least one message is required")
	}
	if len(msgs) > maxQueueBatchMessages {
		return nil, validationError("QueueService.SendBatch", fmt.Sprintf(
			"batch of %d messages exceeds the maximum of %d messages per call; split it into multiple send-batch calls of up to %d messages",
			len(msgs), maxQueueBatchMessages, maxQueueBatchMessages))
	}

	queueID, err := s.resolveQueueID(ctx, queueName)
	if err != nil {
		return nil, err
	}

	for _, m := range msgs {
		warnQueueMessageSize(len(m.Body))
	}

	uri := fmt.Sprintf("/accounts/%s/queues/%s/messages/batch", s.accountID, queueID)
	payload := struct {
		Messages []QueueMessage `json:"messages"`
	}{Messages: msgs}
	raw, err := s.cf.Raw(ctx, http.MethodPost, uri, payload, nil)
	if err != nil {
		return nil, newError("QueueService.SendBatch", fmt.Sprintf("failed to send batch to queue %q", queueName), err)
	}
	if !raw.Success {
		return nil, newError("QueueService.SendBatch", fmt.Sprintf("failed to send batch to queue %q", queueName), nil)
	}

	return &SendBatchResult{Success: true, Sent: len(msgs)}, nil
}

// resolveQueueID resolves a queue name to its queue ID, which the
// messages API requires (Get accepts names, messages endpoints do not).
func (s *QueueService) resolveQueueID(ctx context.Context, queueName string) (string, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	q, err := s.cf.GetQueue(ctx, rc, queueName)
	if err != nil {
		return "", newError("QueueService.resolveQueueID", fmt.Sprintf("failed to resolve queue %q", queueName), err)
	}
	if q.ID == "" {
		return "", validationError("QueueService.resolveQueueID", fmt.Sprintf("queue %q returned no ID", queueName))
	}
	return q.ID, nil
}

// warnQueueMessageSize prints a soft warning when a message body would
// exceed the documented 128 KB (base-10, including ~100 bytes of Queues
// metadata) limit. The message is still sent; the API makes the final call.
func warnQueueMessageSize(bodyLen int) {
	if bodyLen+queueMessageMetadataBytes > maxQueueMessageBytes {
		fmt.Fprintf(os.Stderr,
			"WARNING: message size %d bytes (body %d + ~%d bytes internal Queues metadata) exceeds the %s-byte (128 KB, base-10) Cloudflare Queues limit; the API may reject it\n",
			bodyLen+queueMessageMetadataBytes, bodyLen, queueMessageMetadataBytes, "128,000")
	}
}
