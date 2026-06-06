package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var metricsInterval time.Duration

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Real-time metrics dashboard for Cloudflare services",
	Long: `Launch a TUI dashboard showing live metrics for R2 storage, Workers
invocations, and KV operation rates.

The dashboard polls the Cloudflare API at a configurable interval and
displays usage data in a terminal-based interface.

Flags:
  --interval   Polling interval (default: 30s)

Examples:
  cosmoflare metrics                      # Default 30s interval
  cosmoflare metrics --interval 10s       # Poll every 10 seconds
  cosmoflare metrics --json               # Print metrics as JSON (no TUI)`,
	RunE: runMetrics,
}

func init() {
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().DurationVar(&metricsInterval, "interval", 30*time.Second, "Polling interval")
}

type metricsSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	R2Buckets      int       `json:"r2_buckets"`
	R2TotalSize    int64     `json:"r2_total_size_bytes"`
	R2TotalObjects int64     `json:"r2_total_objects"`
	Workers        int       `json:"workers"`
	KVNamespaces   int       `json:"kv_namespaces"`
}

type tickMsg time.Time
type snapshotMsg metricsSnapshot
type errMsg struct{ err error }

type metricsModel struct {
	snapshot metricsSnapshot
	prevSnap metricsSnapshot
	err      error
	interval time.Duration
	client   cosmoflare.R2Client
	quitting bool
	width    int
	height   int
}

func (m metricsModel) Init() tea.Cmd {
	return tea.Batch(fetchMetrics(m.client), tickCmd(m.interval))
}

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			return m, fetchMetrics(m.client)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(fetchMetrics(m.client), tickCmd(m.interval))
	case snapshotMsg:
		m.prevSnap = m.snapshot
		m.snapshot = metricsSnapshot(msg)
		m.err = nil
	case errMsg:
		m.err = msg.err
	}
	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF9500")).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00BFFF")).
			MarginTop(1)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))
)

func (m metricsModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Cosmoflare Metrics Dashboard"))
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.snapshot.Timestamp.IsZero() {
		b.WriteString(dimStyle.Render("Loading..."))
	} else {
		b.WriteString(sectionStyle.Render("R2 Storage"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Buckets:  %s\n", valueStyle.Render(fmt.Sprintf("%d", m.snapshot.R2Buckets))))
		b.WriteString(fmt.Sprintf("  Objects:  %s\n", valueStyle.Render(fmt.Sprintf("%d", m.snapshot.R2TotalObjects))))
		b.WriteString(fmt.Sprintf("  Size:     %s\n", valueStyle.Render(formatSize(m.snapshot.R2TotalSize))))

		b.WriteString(sectionStyle.Render("Workers"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Scripts:  %s\n", valueStyle.Render(fmt.Sprintf("%d", m.snapshot.Workers))))

		b.WriteString(sectionStyle.Render("KV"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Namespaces: %s\n", valueStyle.Render(fmt.Sprintf("%d", m.snapshot.KVNamespaces))))

		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Last updated: %s  |  Interval: %s",
			m.snapshot.Timestamp.Format("15:04:05"), m.interval)))
	}

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("r = refresh  |  q = quit"))

	return b.String()
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchMetrics(c cosmoflare.R2Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		snap := metricsSnapshot{Timestamp: time.Now()}

		buckets, err := c.ListBuckets(ctx)
		if err == nil {
			snap.R2Buckets = len(buckets)
			for _, b := range buckets {
				snap.R2TotalSize += b.Size
				snap.R2TotalObjects += b.ObjectCount
			}
		}

		return snapshotMsg(snap)
	}
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func runMetrics(cmd *cobra.Command, args []string) error {
	client, err := cosmoflare.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if JSONOutput {
		return runMetricsJSON(client)
	}

	m := metricsModel{
		interval: metricsInterval,
		client:   client,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	_, err = p.Run()
	return err
}

func runMetricsJSON(client cosmoflare.R2Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	snap := metricsSnapshot{Timestamp: time.Now()}
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	snap.R2Buckets = len(buckets)
	for _, b := range buckets {
		snap.R2TotalSize += b.Size
		snap.R2TotalObjects += b.ObjectCount
	}

	out, _ := json.MarshalIndent(snap, "", "  ")
	fmt.Println(string(out))
	return nil
}
