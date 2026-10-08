package cosmoflare

import (
	"context"
	"strings"
	"testing"
)

func TestAnalyticsD1RowsRead(t *testing.T) {
	t.Parallel()
	var cap analyticsCapture
	resp := `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
		`{"sum":{"rowsRead":1000,"rowsWritten":5,"readQueries":10},"dimensions":{"databaseId":"db-small"}},` +
		`{"sum":{"rowsRead":2945546702,"rowsWritten":7,"readQueries":73151},"dimensions":{"databaseId":"db-big"}},` +
		`{"sum":{"rowsRead":500,"rowsWritten":1,"readQueries":2},"dimensions":{"databaseId":"db-small"}}` +
		`]}]}}}`
	srv := analyticsServer(t, resp, &cap)
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.D1RowsRead(context.Background(), analyticsWindow())
	if err != nil {
		t.Fatalf("D1RowsRead: %v", err)
	}
	for _, want := range []string{"d1AnalyticsAdaptiveGroups", "datetime_geq", "rowsRead", "databaseId", "accountTag"} {
		if !strings.Contains(cap.Body, want) {
			t.Errorf("body missing %s: %s", want, cap.Body)
		}
	}
	if strings.Contains(cap.Body, "databaseId:") {
		t.Errorf("query must not filter by databaseId (account-wide fan-out): %s", cap.Body)
	}
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2 (db-small rows summed)", len(got))
	}
	if got[0].DatabaseID != "db-big" || got[0].RowsRead != 2945546702 || got[0].ReadQueries != 73151 {
		t.Errorf("got[0] = %+v, want db-big first (sorted by rows read desc)", got[0])
	}
	if got[1].DatabaseID != "db-small" || got[1].RowsRead != 1500 || got[1].RowsWritten != 6 || got[1].ReadQueries != 12 {
		t.Errorf("got[1] = %+v, want db-small summed 1500/6/12", got[1])
	}
	if got[0].Name != "" {
		t.Errorf("Name = %q, want empty (caller maps names)", got[0].Name)
	}
}

func TestAnalyticsD1RowsReadValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewAnalyticsService("", "tok").D1RowsRead(context.Background(), analyticsWindow()); err == nil {
		t.Error("missing account ID must error")
	}
	w := analyticsWindow()
	w.Start, w.End = w.End, w.Start
	if _, err := NewAnalyticsService("acct", "tok").D1RowsRead(context.Background(), w); err == nil {
		t.Error("inverted window must error")
	}
}
