package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Account represents a single configured Cloudflare account.
type Account struct {
	Name      string `yaml:"name"       json:"name"`
	AccountID string `yaml:"account_id" json:"account_id"`
	APIToken  string `yaml:"api_token"  json:"api_token"`
	Email     string `yaml:"email"      json:"email,omitempty"`
	CreatedAt string `yaml:"created_at" json:"created_at"`
}

// AccountsFile is the on-disk structure for ~/.cosmoflare/accounts.yaml.
type AccountsFile struct {
	Accounts []Account `yaml:"accounts" json:"accounts"`
}

// AccountVerifyResult holds the outcome of a credential verification.
type AccountVerifyResult struct {
	Name    string `json:"name"`
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// AccountService manages multiple Cloudflare accounts stored in
// ~/.cosmoflare/accounts.yaml with an active-account pointer file.
type AccountService struct {
	configDir string

	// verifyFunc calls the Cloudflare API to check credentials.
	// Override in tests to avoid real HTTP calls.
	verifyFunc func(ctx context.Context, accountID, apiToken string) error
}

// NewAccountService creates a new AccountService.
// If configDir is empty, defaults to ~/.cosmoflare.
func NewAccountService(configDir string) (*AccountService, error) {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, newError("AccountService.New", "failed to determine home directory", err)
		}
		configDir = filepath.Join(home, ".cosmoflare")
	}

	return &AccountService{
		configDir:  configDir,
		verifyFunc: defaultVerifyCredentials,
	}, nil
}

// ConfigDir returns the base configuration directory path.
func (s *AccountService) ConfigDir() string {
	return s.configDir
}

// accountsPath returns the path to accounts.yaml.
func (s *AccountService) accountsPath() string {
	return filepath.Join(s.configDir, "accounts.yaml")
}

// activeAccountPath returns the path to the active-account pointer file.
func (s *AccountService) activeAccountPath() string {
	return filepath.Join(s.configDir, "active-account")
}

// loadAccounts reads and parses accounts.yaml, returning an empty list if the
// file does not exist.
func (s *AccountService) loadAccounts() (*AccountsFile, error) {
	data, err := os.ReadFile(s.accountsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &AccountsFile{}, nil
		}
		return nil, newError("AccountService.loadAccounts", "failed to read accounts.yaml", err)
	}

	var af AccountsFile
	if err := yaml.Unmarshal(data, &af); err != nil {
		return nil, newError("AccountService.loadAccounts", "failed to parse accounts.yaml", err)
	}
	return &af, nil
}

// saveAccounts writes accounts.yaml with restricted permissions (0600).
func (s *AccountService) saveAccounts(af *AccountsFile) error {
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return newError("AccountService.saveAccounts", "failed to create config directory", err)
	}

	data, err := yaml.Marshal(af)
	if err != nil {
		return newError("AccountService.saveAccounts", "failed to marshal accounts", err)
	}

	if err := os.WriteFile(s.accountsPath(), data, 0600); err != nil {
		return newError("AccountService.saveAccounts", "failed to write accounts.yaml", err)
	}
	return nil
}

// List returns all configured accounts.
func (s *AccountService) List() ([]Account, error) {
	af, err := s.loadAccounts()
	if err != nil {
		return nil, err
	}
	return af.Accounts, nil
}

// Add registers a new account. The name must be unique and alphanumeric
// (hyphens and underscores allowed). accountID and apiToken are required.
func (s *AccountService) Add(name, accountID, apiToken, email string) (*Account, error) {
	if name == "" {
		return nil, validationError("AccountService.Add", "account name is required")
	}
	if !isValidAccountName(name) {
		return nil, validationError("AccountService.Add",
			fmt.Sprintf("invalid account name %q: only alphanumeric characters, hyphens, and underscores are allowed", name))
	}
	if accountID == "" {
		return nil, validationError("AccountService.Add", "account-id is required")
	}
	if apiToken == "" {
		return nil, validationError("AccountService.Add", "api-token is required")
	}

	af, err := s.loadAccounts()
	if err != nil {
		return nil, err
	}

	// Check for duplicate names
	for _, a := range af.Accounts {
		if a.Name == name {
			return nil, newError("AccountService.Add",
				fmt.Sprintf("account %q already exists (use 'account remove' first)", name), nil)
		}
	}

	acct := Account{
		Name:      name,
		AccountID: accountID,
		APIToken:  apiToken,
		Email:     email,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	af.Accounts = append(af.Accounts, acct)
	if err := s.saveAccounts(af); err != nil {
		return nil, err
	}

	// If this is the first account, make it active automatically
	if len(af.Accounts) == 1 {
		_ = s.writeActiveAccount(name)
	}

	return &acct, nil
}

// Switch sets the active account by name. The account must exist.
func (s *AccountService) Switch(name string) (*Account, error) {
	if name == "" {
		return nil, validationError("AccountService.Switch", "account name is required")
	}

	af, err := s.loadAccounts()
	if err != nil {
		return nil, err
	}

	for _, a := range af.Accounts {
		if a.Name == name {
			if err := s.writeActiveAccount(name); err != nil {
				return nil, err
			}
			return &a, nil
		}
	}

	return nil, newError("AccountService.Switch",
		fmt.Sprintf("account %q not found (run 'account list' to see available accounts)", name), nil)
}

// Remove deletes an account by name. If force is false and the account is
// currently active, an error is returned.
func (s *AccountService) Remove(name string, force bool) error {
	if name == "" {
		return validationError("AccountService.Remove", "account name is required")
	}

	af, err := s.loadAccounts()
	if err != nil {
		return err
	}

	idx := -1
	for i, a := range af.Accounts {
		if a.Name == name {
			idx = i
			break
		}
	}

	if idx == -1 {
		return newError("AccountService.Remove",
			fmt.Sprintf("account %q not found", name), nil)
	}

	// Check if it is the active account
	active, _ := s.readActiveAccount()
	if active == name && !force {
		return newError("AccountService.Remove",
			fmt.Sprintf("account %q is the active account; switch first or use --force", name), nil)
	}

	af.Accounts = append(af.Accounts[:idx], af.Accounts[idx+1:]...)
	if err := s.saveAccounts(af); err != nil {
		return err
	}

	// Clear active-account pointer if we just removed the active one
	if active == name {
		_ = os.Remove(s.activeAccountPath())
	}

	return nil
}

// Current returns the currently active account, or nil if none is set.
func (s *AccountService) Current() (*Account, error) {
	active, err := s.readActiveAccount()
	if err != nil {
		return nil, err
	}
	if active == "" {
		return nil, nil
	}

	af, err := s.loadAccounts()
	if err != nil {
		return nil, err
	}

	for _, a := range af.Accounts {
		if a.Name == active {
			return &a, nil
		}
	}

	// Pointer references a missing account — stale
	return nil, newError("AccountService.Current",
		fmt.Sprintf("active account %q no longer exists in accounts.yaml", active), nil)
}

// Verify checks whether the given account's credentials are valid by calling
// the Cloudflare API.
func (s *AccountService) Verify(name string) (*AccountVerifyResult, error) {
	if name == "" {
		return nil, validationError("AccountService.Verify", "account name is required")
	}

	af, err := s.loadAccounts()
	if err != nil {
		return nil, err
	}

	var acct *Account
	for i, a := range af.Accounts {
		if a.Name == name {
			acct = &af.Accounts[i]
			break
		}
	}
	if acct == nil {
		return nil, newError("AccountService.Verify",
			fmt.Sprintf("account %q not found", name), nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.verifyFunc(ctx, acct.AccountID, acct.APIToken); err != nil {
		return &AccountVerifyResult{
			Name:    name,
			Valid:   false,
			Message: fmt.Sprintf("verification failed: %v", err),
		}, nil
	}

	return &AccountVerifyResult{
		Name:    name,
		Valid:   true,
		Message: "credentials are valid",
	}, nil
}

// writeActiveAccount writes the active account name to the pointer file.
func (s *AccountService) writeActiveAccount(name string) error {
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return newError("AccountService.writeActiveAccount", "failed to create config directory", err)
	}
	if err := os.WriteFile(s.activeAccountPath(), []byte(name), 0600); err != nil {
		return newError("AccountService.writeActiveAccount", "failed to write active-account", err)
	}
	return nil
}

// readActiveAccount reads the pointer file. Returns "" if absent.
func (s *AccountService) readActiveAccount() (string, error) {
	data, err := os.ReadFile(s.activeAccountPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", newError("AccountService.readActiveAccount", "failed to read active-account", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// isValidAccountName checks that a name contains only [a-zA-Z0-9_-].
var accountNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidAccountName(name string) bool {
	return accountNameRe.MatchString(name)
}

// defaultVerifyCredentials calls the Cloudflare /user/tokens/verify endpoint.
func defaultVerifyCredentials(ctx context.Context, accountID, apiToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.cloudflare.com/client/v4/user/tokens/verify", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}
