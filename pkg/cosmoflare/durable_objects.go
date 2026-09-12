package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
)

// DONamespace represents a Durable Objects namespace.
type DONamespace struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Script string `json:"script"`
}

// DOObject represents a Durable Object instance as returned by the
// namespace objects listing endpoint.
type DOObject struct {
	ID            string `json:"id"`
	HasStoredData bool   `json:"has_stored_data"`
}

// DOObjectDetail carries the state of a single Durable Object instance.
//
// The Durable Objects REST API documents no dedicated per-object detail
// endpoint (docs/research/2026-09-10-cf-limits-corpus/coverage-catalog-draft.json,
// "Durable Objects" entry: only namespaces list, namespaces create, objects
// list, script deploy, and GraphQL analytics are documented). GetObject
// locates the object by paging through the objects listing endpoint and
// returns a not-found error if no page contains a matching ID.
type DOObjectDetail struct {
	DOObject
	NamespaceID string `json:"namespace_id"`
}

// ListObjectsOptions controls pagination for ListObjects.
type ListObjectsOptions struct {
	Limit  int
	Cursor string
}

// ListObjectsResult is a single page of Durable Object instances plus the
// cursor needed to fetch the next page (empty when there is no next page).
type ListObjectsResult struct {
	Objects    []DOObject `json:"objects"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

// doListObjectsPageLimit is the page size used internally by GetObject
// when paging through ListObjects to locate a single object.
const doListObjectsPageLimit = 1000

// DurableObjectsService implements Cloudflare Durable Objects live-state
// inspection operations (namespaces and object listing/lookup). It does
// not manage class deployments or bindings — those remain wrangler's job.
type DurableObjectsService struct {
	cf        *cloudflare.API
	accountID string
}

// NewDurableObjectsService creates a new Durable Objects service client.
func NewDurableObjectsService(api *cloudflare.API, accountID string) (*DurableObjectsService, error) {
	if api == nil {
		return nil, validationError("NewDurableObjectsService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewDurableObjectsService", "account ID is required")
	}
	return &DurableObjectsService{cf: api, accountID: accountID}, nil
}

// NewDurableObjectsServiceFromCreds creates a DurableObjectsService from
// account ID and API token.
func NewDurableObjectsServiceFromCreds(accountID, apiToken string) (*DurableObjectsService, error) {
	if accountID == "" {
		return nil, validationError("NewDurableObjectsService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewDurableObjectsService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewDurableObjectsService", "failed to create Cloudflare API client", err)
	}
	return &DurableObjectsService{cf: cf, accountID: accountID}, nil
}

// ListNamespaces returns all Durable Objects namespaces in the account.
func (s *DurableObjectsService) ListNamespaces(ctx context.Context) ([]DONamespace, error) {
	uri := fmt.Sprintf("/accounts/%s/workers/durable_objects/namespaces", s.accountID)
	raw, err := s.cf.Raw(ctx, http.MethodGet, uri, nil, nil)
	if err != nil {
		return nil, newError("DurableObjectsService.ListNamespaces", "failed to list durable object namespaces", err)
	}
	if !raw.Success {
		return nil, newError("DurableObjectsService.ListNamespaces", "failed to list durable object namespaces", nil)
	}

	var namespaces []DONamespace
	if len(raw.Result) > 0 {
		if err := json.Unmarshal(raw.Result, &namespaces); err != nil {
			return nil, newError("DurableObjectsService.ListNamespaces", "failed to decode namespaces response", err)
		}
	}
	return namespaces, nil
}

// ListObjects returns a page of Durable Object instances tracked under a
// namespace, using the API's cursor pagination. opts.Limit caps the page
// size (API default applies when zero); opts.Cursor resumes from a
// previous ListObjectsResult.NextCursor.
func (s *DurableObjectsService) ListObjects(ctx context.Context, namespaceID string, opts ListObjectsOptions) (*ListObjectsResult, error) {
	if namespaceID == "" {
		return nil, validationError("DurableObjectsService.ListObjects", "namespace ID is required")
	}

	uri := fmt.Sprintf("/accounts/%s/workers/durable_objects/namespaces/%s/objects", s.accountID, namespaceID)
	q := url.Values{}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Cursor != "" {
		q.Set("cursor", opts.Cursor)
	}
	if len(q) > 0 {
		uri += "?" + q.Encode()
	}

	raw, err := s.cf.Raw(ctx, http.MethodGet, uri, nil, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, notFound("DurableObjectsService.ListObjects", "", namespaceID, err)
		}
		return nil, newError("DurableObjectsService.ListObjects", fmt.Sprintf("failed to list objects for namespace %q", namespaceID), err)
	}
	if !raw.Success {
		return nil, newError("DurableObjectsService.ListObjects", fmt.Sprintf("failed to list objects for namespace %q", namespaceID), nil)
	}

	var objects []DOObject
	if len(raw.Result) > 0 {
		if err := json.Unmarshal(raw.Result, &objects); err != nil {
			return nil, newError("DurableObjectsService.ListObjects", "failed to decode objects response", err)
		}
	}

	result := &ListObjectsResult{Objects: objects}
	if raw.ResultInfo != nil {
		result.NextCursor = raw.ResultInfo.Cursor
	}
	return result, nil
}

// GetObject locates a single Durable Object instance by ID within a
// namespace. Since the API exposes no per-object detail endpoint, this
// pages through ListObjects (doListObjectsPageLimit per page) until it
// finds a matching ID or exhausts the namespace, returning a not-found
// error in the latter case.
func (s *DurableObjectsService) GetObject(ctx context.Context, namespaceID, objectID string) (*DOObjectDetail, error) {
	if namespaceID == "" {
		return nil, validationError("DurableObjectsService.GetObject", "namespace ID is required")
	}
	if objectID == "" {
		return nil, validationError("DurableObjectsService.GetObject", "object ID is required")
	}

	cursor := ""
	for {
		page, err := s.ListObjects(ctx, namespaceID, ListObjectsOptions{Limit: doListObjectsPageLimit, Cursor: cursor})
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Objects {
			if obj.ID == objectID {
				return &DOObjectDetail{DOObject: obj, NamespaceID: namespaceID}, nil
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	return nil, notFound("DurableObjectsService.GetObject", "", objectID, fmt.Errorf("object %q not found in namespace %q", objectID, namespaceID))
}
