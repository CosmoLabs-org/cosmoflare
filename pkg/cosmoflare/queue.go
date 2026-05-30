package cosmoflare

import (
	"context"
	"fmt"
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
	cf, err := cloudflare.NewWithAPIToken(apiToken)
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
