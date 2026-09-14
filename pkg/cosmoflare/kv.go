package cosmoflare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

// KVNamespace represents a Cloudflare Workers KV namespace.
type KVNamespace struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// KVKey represents a key in a KV namespace.
type KVKey struct {
	Key        string `json:"key"`
	Expiration int64  `json:"expiration,omitempty"`
}

// KVOption is a functional option for KV write operations.
type KVOption func(*kvConfig)

type kvConfig struct {
	ttl      int64
	metadata map[string]string
}

// WithKVTTL sets the expiration TTL in seconds for a KV key.
func WithKVTTL(seconds int64) KVOption {
	return func(c *kvConfig) { c.ttl = seconds }
}

// WithKVMetadata attaches metadata to a KV key.
func WithKVMetadata(m map[string]string) KVOption {
	return func(c *kvConfig) { c.metadata = m }
}

// KVListOption is a functional option for KV list operations.
type KVListOption func(*kvListConfig)

type kvListConfig struct {
	prefix string
	limit  int
	cursor string
}

// WithKVPrefix filters listed keys by the given prefix.
func WithKVPrefix(prefix string) KVListOption {
	return func(c *kvListConfig) { c.prefix = prefix }
}

// WithKVLimit sets the maximum number of keys to return.
func WithKVLimit(n int) KVListOption {
	return func(c *kvListConfig) { c.limit = n }
}

// WithKVCursor sets the pagination cursor.
func WithKVCursor(cursor string) KVListOption {
	return func(c *kvListConfig) { c.cursor = cursor }
}

// KVService implements KV operations.
type KVService struct {
	cf        *cloudflare.API
	accountID string
	rest      *restClient // credential-built services fetch single namespaces directly (BUG-039)
}

// NewKVService creates a new KV service client.
func NewKVService(api *cloudflare.API, accountID string) (*KVService, error) {
	if api == nil {
		return nil, validationError("NewKVService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewKVService", "account ID is required")
	}
	return &KVService{cf: api, accountID: accountID}, nil
}

// NewKVServiceFromCreds creates a KVService from account ID and API token.
func NewKVServiceFromCreds(accountID, apiToken string) (*KVService, error) {
	if accountID == "" {
		return nil, validationError("NewKVService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewKVService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewKVService", "failed to create Cloudflare API client", err)
	}
	rest := newRESTClient(apiToken)
	return &KVService{cf: cf, accountID: accountID, rest: &rest}, nil
}

// CreateNamespace creates a new KV namespace.
func (s *KVService) CreateNamespace(ctx context.Context, title string) (*KVNamespace, error) {
	if title == "" {
		return nil, validationError("KVService.CreateNamespace", "namespace title is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.CreateWorkersKVNamespace(ctx, rc, cloudflare.CreateWorkersKVNamespaceParams{Title: title})
	if err != nil {
		return nil, newError("KVService.CreateNamespace", fmt.Sprintf("failed to create namespace %q", title), err)
	}

	return &KVNamespace{ID: resp.Result.ID, Title: resp.Result.Title}, nil
}

// ListNamespaces returns all KV namespaces in the account.
func (s *KVService) ListNamespaces(ctx context.Context) ([]*KVNamespace, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListWorkersKVNamespaces(ctx, rc, cloudflare.ListWorkersKVNamespacesParams{})
	if err != nil {
		return nil, newError("KVService.ListNamespaces", "failed to list namespaces", err)
	}

	namespaces := make([]*KVNamespace, 0, len(results))
	for _, ns := range results {
		namespaces = append(namespaces, &KVNamespace{ID: ns.ID, Title: ns.Title})
	}
	return namespaces, nil
}

// GetNamespace retrieves a single KV namespace by ID.
//
// Credential-built services fetch the namespace directly
// (GET /accounts/{id}/storage/kv/namespaces/{namespace_id} — one API call;
// BUG-039). Services built via NewKVService carry no token and fall back to
// a list-then-filter scan, which is O(n) in the number of namespaces. For
// batch lookups on the fallback path, call ListNamespaces once and filter
// yourself.
func (s *KVService) GetNamespace(ctx context.Context, id string) (*KVNamespace, error) {
	if id == "" {
		return nil, validationError("KVService.GetNamespace", "namespace ID is required")
	}

	if s.rest != nil {
		var out KVNamespace
		path := fmt.Sprintf("/accounts/%s/storage/kv/namespaces/%s", s.accountID, id)
		if err := s.rest.do(ctx, "KVService.GetNamespace", http.MethodGet, path, nil, &out); err != nil {
			msg := strings.ToLower(err.Error())
			// The API's not-found wording, or a 404 whose body carried no
			// parseable message (rest.do then reports the bare status).
			if strings.Contains(msg, "not found") || strings.Contains(msg, "(http 404)") {
				return nil, notFound("KVService.GetNamespace", "", id, err)
			}
			return nil, err
		}
		if out.ID == "" {
			// A success envelope without the namespace means it does not
			// exist — never fabricate one from the requested ID.
			return nil, notFound("KVService.GetNamespace", "", id, fmt.Errorf("namespace %q not returned", id))
		}
		return &out, nil
	}

	namespaces, err := s.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces {
		if ns.ID == id {
			return ns, nil
		}
	}
	return nil, notFound("KVService.GetNamespace", "", id, fmt.Errorf("namespace %q not found", id))
}

// DeleteNamespace deletes a KV namespace.
func (s *KVService) DeleteNamespace(ctx context.Context, id string) error {
	if id == "" {
		return validationError("KVService.DeleteNamespace", "namespace ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	_, err := s.cf.DeleteWorkersKVNamespace(ctx, rc, id)
	if err != nil {
		return newError("KVService.DeleteNamespace", fmt.Sprintf("failed to delete namespace %q", id), err)
	}
	return nil
}

// Put writes a key-value pair to a KV namespace.
func (s *KVService) Put(ctx context.Context, namespaceID, key string, value io.Reader, opts ...KVOption) error {
	if namespaceID == "" {
		return validationError("KVService.Put", "namespace ID is required")
	}
	if key == "" {
		return validationError("KVService.Put", "key is required")
	}
	if value == nil {
		return validationError("KVService.Put", "value is required")
	}

	cfg := &kvConfig{}
	for _, o := range opts {
		o(cfg)
	}

	data, err := io.ReadAll(value)
	if err != nil {
		return newError("KVService.Put", "failed to read value", err)
	}

	rc := cloudflare.AccountIdentifier(s.accountID)

	if len(cfg.metadata) > 0 || cfg.ttl > 0 {
		pair := &cloudflare.WorkersKVPair{
			Key:   key,
			Value: string(data),
		}
		if len(cfg.metadata) > 0 {
			pair.Metadata = cfg.metadata
		}
		if cfg.ttl > 0 {
			pair.ExpirationTTL = int(cfg.ttl)
		}
		_, err = s.cf.WriteWorkersKVEntries(ctx, rc, cloudflare.WriteWorkersKVEntriesParams{
			NamespaceID: namespaceID,
			KVs:         []*cloudflare.WorkersKVPair{pair},
		})
	} else {
		_, err = s.cf.WriteWorkersKVEntry(ctx, rc, cloudflare.WriteWorkersKVEntryParams{
			NamespaceID: namespaceID,
			Key:         key,
			Value:       data,
		})
	}

	if err != nil {
		return newError("KVService.Put", fmt.Sprintf("failed to write key %q", key), err)
	}
	return nil
}

// Get retrieves a value from a KV namespace.
func (s *KVService) Get(ctx context.Context, namespaceID, key string) ([]byte, error) {
	if namespaceID == "" {
		return nil, validationError("KVService.Get", "namespace ID is required")
	}
	if key == "" {
		return nil, validationError("KVService.Get", "key is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	resp, err := s.cf.GetWorkersKV(ctx, rc, cloudflare.GetWorkersKVParams{
		NamespaceID: namespaceID,
		Key:         key,
	})
	if err != nil {
		return nil, newError("KVService.Get", fmt.Sprintf("failed to get key %q", key), err)
	}

	return resp, nil
}

// Delete removes a key from a KV namespace.
func (s *KVService) Delete(ctx context.Context, namespaceID, key string) error {
	if namespaceID == "" {
		return validationError("KVService.Delete", "namespace ID is required")
	}
	if key == "" {
		return validationError("KVService.Delete", "key is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	_, err := s.cf.DeleteWorkersKVEntry(ctx, rc, cloudflare.DeleteWorkersKVEntryParams{
		NamespaceID: namespaceID,
		Key:         key,
	})
	if err != nil {
		return newError("KVService.Delete", fmt.Sprintf("failed to delete key %q", key), err)
	}
	return nil
}

// ListKeys returns keys in a KV namespace with optional filtering.
func (s *KVService) ListKeys(ctx context.Context, namespaceID string, opts ...KVListOption) (*ListResult[*KVKey], error) {
	if namespaceID == "" {
		return nil, validationError("KVService.ListKeys", "namespace ID is required")
	}

	cfg := &kvListConfig{}
	for _, o := range opts {
		o(cfg)
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.ListWorkersKVsParams{
		NamespaceID: namespaceID,
		Prefix:      cfg.prefix,
		Limit:       cfg.limit,
		Cursor:      cfg.cursor,
	}

	resp, err := s.cf.ListWorkersKVKeys(ctx, rc, params)
	if err != nil {
		return nil, newError("KVService.ListKeys", "failed to list keys", err)
	}

	keys := make([]*KVKey, 0, len(resp.Result))
	for _, k := range resp.Result {
		keys = append(keys, &KVKey{
			Key:        k.Name,
			Expiration: int64(k.Expiration),
		})
	}

	result := &ListResult[*KVKey]{
		Items:       keys,
		IsTruncated: resp.ResultInfo.Cursors.Before != "" || resp.ResultInfo.Cursors.After != "",
	}
	if resp.ResultInfo.Cursors.After != "" {
		result.NextToken = resp.ResultInfo.Cursors.After
	}

	return result, nil
}
