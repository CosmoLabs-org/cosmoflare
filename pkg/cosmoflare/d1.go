package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// D1Database represents a Cloudflare D1 database.
type D1Database struct {
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	NumTables int        `json:"num_tables"`
	FileSize  int64      `json:"file_size"`
	CreatedAt *time.Time `json:"created_at"`
}

// D1QueryResult represents the result of a D1 SQL query.
type D1QueryResult struct {
	Columns []string          `json:"columns"`
	Rows    []map[string]any  `json:"rows"`
	Meta    D1QueryMeta       `json:"meta"`
	Success bool              `json:"success"`
}

// D1QueryMeta contains metadata about a query execution.
type D1QueryMeta struct {
	ChangedDB   bool    `json:"changed_db"`
	Changes     int     `json:"changes"`
	Duration    float64 `json:"duration"`
	LastRowID   int     `json:"last_row_id"`
	RowsRead    int     `json:"rows_read"`
	RowsWritten int     `json:"rows_written"`
	SizeAfter   int     `json:"size_after"`
}

// D1Service implements D1 database operations.
type D1Service struct {
	cf        *cloudflare.API
	accountID string
}

// NewD1Service creates a new D1 service client.
func NewD1Service(api *cloudflare.API, accountID string) (*D1Service, error) {
	if api == nil {
		return nil, validationError("NewD1Service", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewD1Service", "account ID is required")
	}
	return &D1Service{cf: api, accountID: accountID}, nil
}

// NewD1ServiceFromCreds creates a D1Service from account ID and API token.
func NewD1ServiceFromCreds(accountID, apiToken string) (*D1Service, error) {
	if accountID == "" {
		return nil, validationError("NewD1Service", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewD1Service", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewD1Service", "failed to create Cloudflare API client", err)
	}
	return &D1Service{cf: cf, accountID: accountID}, nil
}

// Create creates a new D1 database.
func (s *D1Service) Create(ctx context.Context, name string) (*D1Database, error) {
	if name == "" {
		return nil, validationError("D1Service.Create", "database name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.CreateD1Database(ctx, rc, cloudflare.CreateD1DatabaseParams{Name: name})
	if err != nil {
		return nil, newError("D1Service.Create", fmt.Sprintf("failed to create database %q", name), err)
	}

	return mapD1Database(result), nil
}

// List returns all D1 databases in the account.
func (s *D1Service) List(ctx context.Context) ([]*D1Database, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListD1Databases(ctx, rc, cloudflare.ListD1DatabasesParams{})
	if err != nil {
		return nil, newError("D1Service.List", "failed to list databases", err)
	}

	databases := make([]*D1Database, 0, len(results))
	for _, db := range results {
		databases = append(databases, mapD1Database(db))
	}
	return databases, nil
}

// Get retrieves a single D1 database by ID.
func (s *D1Service) Get(ctx context.Context, databaseID string) (*D1Database, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.Get", "database ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetD1Database(ctx, rc, databaseID)
	if err != nil {
		return nil, newError("D1Service.Get", fmt.Sprintf("failed to get database %q", databaseID), err)
	}

	return mapD1Database(result), nil
}

// Delete deletes a D1 database.
func (s *D1Service) Delete(ctx context.Context, databaseID string) error {
	if databaseID == "" {
		return validationError("D1Service.Delete", "database ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeleteD1Database(ctx, rc, databaseID)
	if err != nil {
		return newError("D1Service.Delete", fmt.Sprintf("failed to delete database %q", databaseID), err)
	}
	return nil
}

// Query executes a SQL query against a D1 database.
func (s *D1Service) Query(ctx context.Context, databaseID, sql string, params ...string) ([]*D1QueryResult, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.Query", "database ID is required")
	}
	if sql == "" {
		return nil, validationError("D1Service.Query", "SQL query is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	results, err := s.cf.QueryD1Database(ctx, rc, cloudflare.QueryD1DatabaseParams{
		DatabaseID: databaseID,
		SQL:        sql,
		Parameters: params,
	})
	if err != nil {
		return nil, newError("D1Service.Query", fmt.Sprintf("failed to query database %q", databaseID), err)
	}

	queryResults := make([]*D1QueryResult, 0, len(results))
	for _, r := range results {
		queryResults = append(queryResults, mapD1Result(r))
	}
	return queryResults, nil
}

// mapD1Database converts a cloudflare.D1Database to our D1Database type.
func mapD1Database(db cloudflare.D1Database) *D1Database {
	return &D1Database{
		UUID:      db.UUID,
		Name:      db.Name,
		Version:   db.Version,
		NumTables: db.NumTables,
		FileSize:  db.FileSize,
		CreatedAt: db.CreatedAt,
	}
}

// mapD1Result converts a cloudflare.D1Result to our D1QueryResult type.
func mapD1Result(r cloudflare.D1Result) *D1QueryResult {
	success := false
	if r.Success != nil {
		success = *r.Success
	}

	changedDB := false
	if r.Meta.ChangedDB != nil {
		changedDB = *r.Meta.ChangedDB
	}

	return &D1QueryResult{
		Rows:    r.Results,
		Success: success,
		Meta: D1QueryMeta{
			ChangedDB:   changedDB,
			Changes:     r.Meta.Changes,
			Duration:    r.Meta.Duration,
			LastRowID:   r.Meta.LastRowID,
			RowsRead:    r.Meta.RowsRead,
			RowsWritten: r.Meta.RowsWritten,
			SizeAfter:   r.Meta.SizeAfter,
		},
	}
}
