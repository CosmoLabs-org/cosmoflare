package cosmoflare

import (
	"context"
	"sort"
	"time"
)

// D1RowsReadSummary is one database's D1 query load over a window (FEAT-049).
// D1 bills rows read (scanned), so RowsRead is the waste signal.
type D1RowsReadSummary struct {
	DatabaseID  string `json:"database_id"`
	Name        string `json:"name,omitempty"` // filled by callers from the D1 list; empty when unknown
	RowsRead    uint64 `json:"rows_read"`
	RowsWritten uint64 `json:"rows_written"`
	ReadQueries uint64 `json:"read_queries"`
}

// D1RowsRead returns per-database rows read/written over the window in one
// account-wide query grouped by databaseId (no per-database N+1). Rows are
// sorted by RowsRead descending, then DatabaseID.
func (s *AnalyticsService) D1RowsRead(ctx context.Context, w AnalyticsWindow) ([]D1RowsReadSummary, error) {
	const op = "AnalyticsD1RowsRead"
	if err := s.validateAccount(op); err != nil {
		return nil, err
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($accountTag: String!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      d1AnalyticsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        sum { rowsRead rowsWritten readQueries }
        dimensions { databaseId }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				Groups []struct {
					Sum struct {
						RowsRead    uint64 `json:"rowsRead"`
						RowsWritten uint64 `json:"rowsWritten"`
						ReadQueries uint64 `json:"readQueries"`
					} `json:"sum"`
					Dimensions struct {
						DatabaseID string `json:"databaseId"`
					} `json:"dimensions"`
				} `json:"d1AnalyticsAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, op, gql, s.windowVars(w), &out); err != nil {
		return nil, err
	}
	byID := make(map[string]*D1RowsReadSummary)
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.Groups {
			row := byID[g.Dimensions.DatabaseID]
			if row == nil {
				row = &D1RowsReadSummary{DatabaseID: g.Dimensions.DatabaseID}
				byID[g.Dimensions.DatabaseID] = row
			}
			row.RowsRead += g.Sum.RowsRead
			row.RowsWritten += g.Sum.RowsWritten
			row.ReadQueries += g.Sum.ReadQueries
		}
	}
	rows := make([]D1RowsReadSummary, 0, len(byID))
	for _, r := range byID {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].RowsRead != rows[j].RowsRead {
			return rows[i].RowsRead > rows[j].RowsRead
		}
		return rows[i].DatabaseID < rows[j].DatabaseID
	})
	return rows, nil
}
