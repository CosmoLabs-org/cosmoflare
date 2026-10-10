package cosmoflare

import (
	"context"
	"sort"
	"time"
)

// KVWritesSummary is one namespace's KV write volume over a window (FEAT-066,
// parity with the Ops worker's kv-writes condition). Only actionType "write"
// rows count: reads are a separate signal and deletes are unmetered upstream.
type KVWritesSummary struct {
	NamespaceID string `json:"namespace_id"`
	Name        string `json:"name,omitempty"` // filled by callers from the namespace list; empty when unknown
	Writes      uint64 `json:"writes"`
}

// KVWrites returns per-namespace KV write volume over the window in one
// account-wide query grouped by namespaceId + actionType (the dataset the
// billing collector already reads; verified live 2026-10-10). Read and delete
// rows are dropped here, not upstream. Rows are sorted by Writes descending,
// then NamespaceID.
func (s *AnalyticsService) KVWrites(ctx context.Context, w AnalyticsWindow) ([]KVWritesSummary, error) {
	const op = "AnalyticsKVWrites"
	if err := s.validateAccount(op); err != nil {
		return nil, err
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($accountTag: String!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      kvOperationsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        sum { requests }
        dimensions { namespaceId actionType }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				Groups []struct {
					Sum struct {
						Requests uint64 `json:"requests"`
					} `json:"sum"`
					Dimensions struct {
						NamespaceID string `json:"namespaceId"`
						ActionType  string `json:"actionType"`
					} `json:"dimensions"`
				} `json:"kvOperationsAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, op, gql, s.windowVars(w), &out); err != nil {
		return nil, err
	}
	byID := make(map[string]*KVWritesSummary)
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.Groups {
			if g.Dimensions.ActionType != "write" {
				continue
			}
			row := byID[g.Dimensions.NamespaceID]
			if row == nil {
				row = &KVWritesSummary{NamespaceID: g.Dimensions.NamespaceID}
				byID[g.Dimensions.NamespaceID] = row
			}
			row.Writes += g.Sum.Requests
		}
	}
	rows := make([]KVWritesSummary, 0, len(byID))
	for _, r := range byID {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Writes != rows[j].Writes {
			return rows[i].Writes > rows[j].Writes
		}
		return rows[i].NamespaceID < rows[j].NamespaceID
	})
	return rows, nil
}
