package cosmoflare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// AlertService provides CRUD operations for alert rules and history tracking.
// Alert rules are stored in .cosmoflare-alerts.yaml in the project root.
// Alert history is stored in ~/.cosmoflare/alert-history.log (NDJSON).
type AlertService struct {
	rulesPath   string // path to .cosmoflare-alerts.yaml
	historyPath string // path to alert-history.log
	mu          sync.RWMutex
}

// AlertRule defines a single alerting rule configuration.
type AlertRule struct {
	Name      string    `json:"name" yaml:"name"`
	Service   string    `json:"service" yaml:"service"`       // r2, workers, kv, dns
	Condition string    `json:"condition" yaml:"condition"`    // error-rate, storage-limit, latency, failure-count, workers-script-count, r2-bucket-count, dns-record-quota
	Threshold float64   `json:"threshold" yaml:"threshold"`
	Action    string    `json:"action" yaml:"action"`          // webhook, email, log
	Target    string    `json:"target" yaml:"target"`          // URL or email address
	Enabled   bool      `json:"enabled" yaml:"enabled"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// AlertHistory records a single alert trigger event.
type AlertHistory struct {
	RuleName  string    `json:"rule_name"`
	Service   string    `json:"service"`
	Condition string    `json:"condition"`
	Threshold float64   `json:"threshold"`
	Value     float64   `json:"value"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	IsTest    bool      `json:"is_test,omitempty"`
}

// AlertsConfig is the top-level structure for .cosmoflare-alerts.yaml.
type AlertsConfig struct {
	Version string       `json:"version" yaml:"version"`
	Rules   []*AlertRule `json:"rules" yaml:"rules"`
}

// Valid services for alert rules.
var validAlertServices = map[string]bool{
	"r2":      true,
	"workers": true,
	"kv":      true,
	"dns":     true,
}

// AlertConditionDescriptor describes one alertable condition — the single
// registry every surface derives from (FEAT-015): validation, error
// messages, CLI help text, and the evaluator's data-key mapping. Adding a
// condition here plus its case in the evaluator's conditionValue makes it
// valid everywhere; extending one place and not the others is the drift this
// table exists to prevent.
type AlertConditionDescriptor struct {
	Name    string // rule condition key
	Unit    string // display unit of the observed value
	Help    string // one-line description for help output
	FedBy   string // availability rule: which metric source feeds it
	DataKey string // EvalMetrics field the evaluator reads
	Service string // service tag for grouping in help output
}

// alertConditionRegistry is the ordered condition registry. The three
// count/quota conditions are fed by the serve alert cycle's LimitsService
// snapshot (CollectLimitMetrics).
var alertConditionRegistry = []AlertConditionDescriptor{
	{Name: "error-rate", Unit: "%", Help: "share of Workers requests that errored over the window", FedBy: "Workers analytics (requests > 0)", DataKey: "WorkersErrors/WorkersRequests", Service: "workers"},
	{Name: "storage-limit", Unit: "bytes", Help: "R2 storage in use", FedBy: "R2 analytics", DataKey: "R2StorageBytes", Service: "r2"},
	{Name: "latency", Unit: "ms", Help: "Workers CPU p99 average", FedBy: "Workers analytics (CPU samples > 0)", DataKey: "CPUP99AvgMS", Service: "workers"},
	{Name: "failure-count", Unit: "errors", Help: "absolute Workers error count over the window", FedBy: "Workers analytics", DataKey: "WorkersErrors", Service: "workers"},
	{Name: "workers-script-count", Unit: "scripts", Help: "Workers scripts on the account against the plan limit", FedBy: "limits snapshot", DataKey: "WorkersScriptCount", Service: "workers"},
	{Name: "r2-bucket-count", Unit: "buckets", Help: "R2 buckets on the account against the plan limit", FedBy: "limits snapshot", DataKey: "R2BucketCount", Service: "r2"},
	{Name: "dns-record-quota", Unit: "%", Help: "highest per-zone DNS record usage against the zone quota", FedBy: "limits snapshot (DNS rows)", DataKey: "DNSRecordQuotaPct", Service: "dns"},
}

// AlertConditions returns the condition registry in registration order.
func AlertConditions() []AlertConditionDescriptor {
	out := make([]AlertConditionDescriptor, len(alertConditionRegistry))
	copy(out, alertConditionRegistry)
	return out
}

// AlertConditionList renders the registered condition names for error
// messages and help text.
func AlertConditionList() string {
	names := make([]string, len(alertConditionRegistry))
	for i, c := range alertConditionRegistry {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

// validAlertCondition reports whether name is a registered condition.
func validAlertCondition(name string) bool {
	for _, c := range alertConditionRegistry {
		if c.Name == name {
			return true
		}
	}
	return false
}

// Valid actions for alert rules.
var validAlertActions = map[string]bool{
	"webhook": true,
	"email":   true,
	"log":     true,
}

// NewAlertService creates a new AlertService with default paths.
// rulesPath defaults to .cosmoflare-alerts.yaml in the current directory.
// historyPath defaults to ~/.cosmoflare/alert-history.log.
func NewAlertService(rulesPath, historyPath string) (*AlertService, error) {
	if rulesPath == "" {
		rulesPath = ".cosmoflare-alerts.yaml"
	}
	if historyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, newError("NewAlertService", "failed to determine home directory", err)
		}
		historyPath = filepath.Join(home, ".cosmoflare", "alert-history.log")
	}
	return &AlertService{
		rulesPath:   rulesPath,
		historyPath: historyPath,
	}, nil
}

// loadConfig reads and parses the alerts config file.
// Returns an empty config if the file does not exist.
func (s *AlertService) loadConfig() (*AlertsConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.rulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AlertsConfig{Version: "1", Rules: []*AlertRule{}}, nil
		}
		return nil, newError("AlertService.loadConfig", "failed to read alerts config", err)
	}

	var cfg AlertsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, newError("AlertService.loadConfig", "failed to parse alerts config", err)
	}
	if cfg.Rules == nil {
		cfg.Rules = []*AlertRule{}
	}
	if cfg.Version == "" {
		cfg.Version = "1"
	}
	return &cfg, nil
}

// saveConfig writes the alerts config to disk.
func (s *AlertService) saveConfig(cfg *AlertsConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return newError("AlertService.saveConfig", "failed to marshal alerts config", err)
	}
	if err := os.WriteFile(s.rulesPath, data, 0644); err != nil {
		return newError("AlertService.saveConfig", "failed to write alerts config", err)
	}
	return nil
}

// List returns all configured alert rules.
func (s *AlertService) List() ([]*AlertRule, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Rules, nil
}

// Get retrieves a single alert rule by name.
func (s *AlertService) Get(name string) (*AlertRule, error) {
	if name == "" {
		return nil, validationError("AlertService.Get", "alert rule name is required")
	}

	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}

	for _, r := range cfg.Rules {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, notFound("AlertService.Get", "", name, nil)
}

// Create adds a new alert rule. Returns an error if a rule with the same name already exists.
func (s *AlertService) Create(rule *AlertRule) (*AlertRule, error) {
	if err := validateAlertRule(rule); err != nil {
		return nil, err
	}

	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}

	// Check for duplicate name
	for _, r := range cfg.Rules {
		if r.Name == rule.Name {
			return nil, validationError("AlertService.Create", fmt.Sprintf("alert rule %q already exists", rule.Name))
		}
	}

	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	rule.Enabled = true

	cfg.Rules = append(cfg.Rules, rule)
	if err := s.saveConfig(cfg); err != nil {
		return nil, err
	}
	return rule, nil
}

// Update modifies an existing alert rule. Only non-zero fields on the update are applied.
func (s *AlertService) Update(name string, update *AlertRule) (*AlertRule, error) {
	if name == "" {
		return nil, validationError("AlertService.Update", "alert rule name is required")
	}

	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}

	var found *AlertRule
	for _, r := range cfg.Rules {
		if r.Name == name {
			found = r
			break
		}
	}
	if found == nil {
		return nil, notFound("AlertService.Update", "", name, nil)
	}

	// Apply non-zero updates
	if update.Service != "" {
		if !validAlertServices[update.Service] {
			return nil, validationError("AlertService.Update", fmt.Sprintf("invalid service %q, must be one of: r2, workers, kv, dns", update.Service))
		}
		found.Service = update.Service
	}
	if update.Condition != "" {
		if !validAlertCondition(update.Condition) {
			return nil, validationError("AlertService.Update", fmt.Sprintf("invalid condition %q, must be one of: %s", update.Condition, AlertConditionList()))
		}
		found.Condition = update.Condition
	}
	if update.Threshold != 0 {
		found.Threshold = update.Threshold
	}
	if update.Action != "" {
		if !validAlertActions[update.Action] {
			return nil, validationError("AlertService.Update", fmt.Sprintf("invalid action %q, must be one of: webhook, email, log", update.Action))
		}
		found.Action = update.Action
	}
	if update.Target != "" {
		found.Target = update.Target
	}

	found.UpdatedAt = time.Now()

	if err := s.saveConfig(cfg); err != nil {
		return nil, err
	}
	return found, nil
}

// Delete removes an alert rule by name.
func (s *AlertService) Delete(name string) error {
	if name == "" {
		return validationError("AlertService.Delete", "alert rule name is required")
	}

	cfg, err := s.loadConfig()
	if err != nil {
		return err
	}

	idx := -1
	for i, r := range cfg.Rules {
		if r.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return notFound("AlertService.Delete", "", name, nil)
	}

	cfg.Rules = append(cfg.Rules[:idx], cfg.Rules[idx+1:]...)
	return s.saveConfig(cfg)
}

// Test fires a test alert for the named rule and records it in history.
func (s *AlertService) Test(name string) (*AlertHistory, error) {
	rule, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	entry := &AlertHistory{
		RuleName:  rule.Name,
		Service:   rule.Service,
		Condition: rule.Condition,
		Threshold: rule.Threshold,
		Value:     rule.Threshold, // simulate threshold-met
		Action:    rule.Action,
		Target:    rule.Target,
		Message:   fmt.Sprintf("Test alert for rule %q", rule.Name),
		Timestamp: time.Now(),
		IsTest:    true,
	}

	if err := s.appendHistory(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Evaluate checks a metric value against a named rule. If the threshold is met,
// it records a history entry and returns it. Returns nil if the threshold is not met.
func (s *AlertService) Evaluate(name string, currentValue float64) (*AlertHistory, error) {
	rule, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	if !rule.Enabled {
		return nil, nil
	}

	triggered := false
	switch rule.Condition {
	case "error-rate", "latency", "failure-count":
		triggered = currentValue >= rule.Threshold
	case "storage-limit":
		triggered = currentValue >= rule.Threshold
	}

	if !triggered {
		return nil, nil
	}

	entry := &AlertHistory{
		RuleName:  rule.Name,
		Service:   rule.Service,
		Condition: rule.Condition,
		Threshold: rule.Threshold,
		Value:     currentValue,
		Action:    rule.Action,
		Target:    rule.Target,
		Message:   fmt.Sprintf("Alert triggered: %s %s %.2f >= %.2f", rule.Service, rule.Condition, currentValue, rule.Threshold),
		Timestamp: time.Now(),
	}

	if err := s.appendHistory(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// History reads alert history entries from the NDJSON log file.
// limit <= 0 returns all entries. since filters entries after the given time.
func (s *AlertService) History(limit int, since time.Time) ([]*AlertHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AlertHistory{}, nil
		}
		return nil, newError("AlertService.History", "failed to read history file", err)
	}

	var entries []*AlertHistory
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry AlertHistory
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}
		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}
		entries = append(entries, &entry)
	}

	// Return most recent entries if limit is set
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// appendHistory writes a single history entry as NDJSON.
func (s *AlertService) appendHistory(entry *AlertHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure parent directory exists
	dir := filepath.Dir(s.historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return newError("AlertService.appendHistory", "failed to create history directory", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return newError("AlertService.appendHistory", "failed to marshal history entry", err)
	}

	f, err := os.OpenFile(s.historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return newError("AlertService.appendHistory", "failed to open history file", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return newError("AlertService.appendHistory", "failed to write history entry", err)
	}
	return nil
}

// validateAlertRule checks that all required fields are present and valid.
func validateAlertRule(rule *AlertRule) error {
	if rule == nil {
		return validationError("validateAlertRule", "alert rule is required")
	}
	if rule.Name == "" {
		return validationError("validateAlertRule", "alert rule name is required")
	}
	if !validAlertServices[rule.Service] {
		return validationError("validateAlertRule", fmt.Sprintf("invalid service %q, must be one of: r2, workers, kv, dns", rule.Service))
	}
	if !validAlertCondition(rule.Condition) {
		return validationError("validateAlertRule", fmt.Sprintf("invalid condition %q, must be one of: %s", rule.Condition, AlertConditionList()))
	}
	if rule.Threshold <= 0 {
		return validationError("validateAlertRule", "threshold must be a positive number")
	}
	if !validAlertActions[rule.Action] {
		return validationError("validateAlertRule", fmt.Sprintf("invalid action %q, must be one of: webhook, email, log", rule.Action))
	}
	if rule.Target == "" {
		return validationError("validateAlertRule", "target is required (URL for webhook, email address for email, path for log)")
	}
	return nil
}

// splitLines splits byte data into lines without importing bufio.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
