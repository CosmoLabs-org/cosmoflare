package cosmoflare

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// D1UsageMaxDays bounds the --days window. Cloudflare's GraphQL analytics
// for D1 retains daily rows for up to three months; anything beyond that
// would silently truncate, so it is rejected up front.
const D1UsageMaxDays = 92

// D1DailyUsage is rows read and written for one database on one UTC day.
// Date is formatted YYYY-MM-DD, matching the GraphQL date dimension.
type D1DailyUsage struct {
	Date        string `json:"date"`
	RowsRead    int64  `json:"rows_read"`
	RowsWritten int64  `json:"rows_written"`
}

// D1UsageReport combines the daily usage window with the database's
// point-in-time facts (file size, table count) so commands can render a
// complete health picture from a single call.
type D1UsageReport struct {
	Days             []D1DailyUsage `json:"days"`
	TotalRowsRead    int64          `json:"total_rows_read"`
	TotalRowsWritten int64          `json:"total_rows_written"`
	FileSizeBytes    int64          `json:"file_size_bytes"`
	NumTables        int            `json:"num_tables"`
}

// D1UsageService reads D1 daily usage from Cloudflare's GraphQL Analytics
// API (d1AnalyticsAdaptiveGroups) and database facts from the REST D1 API.
// Since 2026-09-01 Cloudflare enforces free-tier D1 daily limits by failing
// queries, so this visibility is outage prevention, not decoration.
type D1UsageService struct {
	analytics *AnalyticsService
	d1        *D1Service

	// nowFn is a test-only seam for the UTC "today" anchor of the daily
	// window. Defaults to time.Now; set directly by tests in this package,
	// mirroring D1Service.sleepFn.
	nowFn func() time.Time
}

// NewD1UsageService creates a D1 usage service from an analytics service
// (GraphQL) and a D1 service (REST). Both are required.
func NewD1UsageService(analytics *AnalyticsService, d1 *D1Service) (*D1UsageService, error) {
	if analytics == nil {
		return nil, validationError("NewD1UsageService", "analytics service is required")
	}
	if d1 == nil {
		return nil, validationError("NewD1UsageService", "D1 service is required")
	}
	return &D1UsageService{analytics: analytics, d1: d1, nowFn: time.Now}, nil
}

// NewD1UsageServiceFromCreds builds the underlying analytics and D1 services
// from account ID and API token. Empty credentials surface as validation
// errors so command runners can fail fast before any network call.
func NewD1UsageServiceFromCreds(accountID, apiToken string) (*D1UsageService, error) {
	d1, err := NewD1ServiceFromCreds(accountID, apiToken)
	if err != nil {
		return nil, err
	}
	return NewD1UsageService(NewAnalyticsService(accountID, apiToken), d1)
}

// DailyUsage returns per-day rows read/written for one database over the
// trailing days UTC days (today included). The GraphQL window is exclusive
// of tomorrow so "today" is fully covered, and the API's date_geq/date_lt
// filters do the date scoping server-side.
func (s *D1UsageService) DailyUsage(ctx context.Context, databaseID string, days int) ([]D1DailyUsage, error) {
	const op = "D1UsageService.DailyUsage"
	if databaseID == "" {
		return nil, validationError(op, "database ID is required")
	}
	if days < 1 || days > D1UsageMaxDays {
		return nil, validationError(op, fmt.Sprintf("days must be between 1 and %d, got %d", D1UsageMaxDays, days))
	}

	now := s.nowFn().UTC()
	start := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	end := now.AddDate(0, 0, 1).Format("2006-01-02")

	gql := `query($accountTag: String!, $databaseId: String!, $start: Date!, $end: Date!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      d1AnalyticsAdaptiveGroups(limit: 10000, filter: {date_geq: $start, date_lt: $end, databaseId: $databaseId}) {
        sum { rowsRead rowsWritten }
        dimensions { date databaseId }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				D1AnalyticsAdaptiveGroups []struct {
					Sum struct {
						RowsRead    int64 `json:"rowsRead"`
						RowsWritten int64 `json:"rowsWritten"`
					} `json:"sum"`
					Dimensions struct {
						Date       string `json:"date"`
						DatabaseID string `json:"databaseId"`
					} `json:"dimensions"`
				} `json:"d1AnalyticsAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	vars := map[string]any{
		"accountTag": s.d1.accountID,
		"databaseId": databaseID,
		"start":      start,
		"end":        end,
	}
	if err := s.analytics.query(ctx, op, gql, vars, &out); err != nil {
		return nil, err
	}

	byDate := map[string]*D1DailyUsage{}
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.D1AnalyticsAdaptiveGroups {
			day, ok := byDate[g.Dimensions.Date]
			if !ok {
				day = &D1DailyUsage{Date: g.Dimensions.Date}
				byDate[g.Dimensions.Date] = day
			}
			day.RowsRead += g.Sum.RowsRead
			day.RowsWritten += g.Sum.RowsWritten
		}
	}

	usage := make([]D1DailyUsage, 0, len(byDate))
	for _, day := range byDate {
		usage = append(usage, *day)
	}
	sort.Slice(usage, func(i, j int) bool { return usage[i].Date < usage[j].Date })
	return usage, nil
}

// Report returns the daily usage window plus the database's current file
// size and table count (via D1Service.Get).
func (s *D1UsageService) Report(ctx context.Context, databaseID string, days int) (*D1UsageReport, error) {
	const op = "D1UsageService.Report"
	usage, err := s.DailyUsage(ctx, databaseID, days)
	if err != nil {
		return nil, err
	}
	db, err := s.d1.Get(ctx, databaseID)
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to get database %q", databaseID), err)
	}

	report := &D1UsageReport{
		Days:          usage,
		FileSizeBytes: db.FileSize,
		NumTables:     db.NumTables,
	}
	for _, day := range usage {
		report.TotalRowsRead += day.RowsRead
		report.TotalRowsWritten += day.RowsWritten
	}
	return report, nil
}

// Today returns today's UTC usage, or a zero entry when analytics has no
// row for today yet (a database that has not been queried today).
func (r *D1UsageReport) Today() D1DailyUsage {
	today := time.Now().UTC().Format("2006-01-02")
	// Days are sorted ascending by DailyUsage, so scan from the end.
	for i := len(r.Days) - 1; i >= 0; i-- {
		if r.Days[i].Date == today {
			return r.Days[i]
		}
	}
	return D1DailyUsage{Date: today}
}
