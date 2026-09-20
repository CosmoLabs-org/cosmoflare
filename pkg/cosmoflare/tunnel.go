package cosmoflare

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Tunnel represents a Cloudflare Tunnel (cfd_tunnel) in the account.
type Tunnel struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	CreatedAt    *time.Time         `json:"created_at,omitempty"`
	DeletedAt    *time.Time         `json:"deleted_at,omitempty"`
	Status       string             `json:"status,omitempty"`
	TunnelType   string             `json:"tunnel_type,omitempty"`
	RemoteConfig bool               `json:"remote_config,omitempty"`
	Connections  []TunnelConnection `json:"connections,omitempty"`
}

// TunnelConnection represents a single edge connection of a connector.
type TunnelConnection struct {
	ID                 string `json:"id"`
	ColoName           string `json:"colo_name"`
	ClientID           string `json:"client_id"`
	ClientVersion      string `json:"client_version"`
	OpenedAt           string `json:"opened_at"`
	OriginIP           string `json:"origin_ip"`
	IsPendingReconnect bool   `json:"is_pending_reconnect"`
}

// TunnelConnector represents a cloudflared instance connected to a tunnel,
// together with its edge connections.
type TunnelConnector struct {
	ID          string             `json:"id"`
	Features    []string           `json:"features,omitempty"`
	Version     string             `json:"version,omitempty"`
	Arch        string             `json:"arch,omitempty"`
	RunAt       *time.Time         `json:"run_at,omitempty"`
	Connections []TunnelConnection `json:"connections,omitempty"`
}

// tunnelIDPattern matches the UUID-shaped identifiers Cloudflare assigns to
// tunnels, so callers can pass either an ID or a human name.
var tunnelIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// TunnelService implements Cloudflare Tunnel operations.
type TunnelService struct {
	cf        *cloudflare.API
	accountID string
}

// NewTunnelService creates a new Tunnel service client.
func NewTunnelService(api *cloudflare.API, accountID string) (*TunnelService, error) {
	if api == nil {
		return nil, validationError("NewTunnelService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewTunnelService", "account ID is required")
	}
	return &TunnelService{cf: api, accountID: accountID}, nil
}

// NewTunnelServiceFromCreds creates a TunnelService from account ID and API token.
func NewTunnelServiceFromCreds(accountID, apiToken string) (*TunnelService, error) {
	if accountID == "" {
		return nil, validationError("NewTunnelService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewTunnelService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewTunnelService", "failed to create Cloudflare API client", err)
	}
	return &TunnelService{cf: cf, accountID: accountID}, nil
}

// List returns all tunnels in the account.
func (s *TunnelService) List(ctx context.Context) ([]*Tunnel, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListTunnels(ctx, rc, cloudflare.TunnelListParams{})
	if err != nil {
		return nil, newError("TunnelService.List", "failed to list tunnels", err)
	}

	tunnels := make([]*Tunnel, 0, len(results))
	for _, t := range results {
		tunnels = append(tunnels, mapTunnel(t))
	}
	return tunnels, nil
}

// Get retrieves a single tunnel by ID.
func (s *TunnelService) Get(ctx context.Context, tunnelID string) (*Tunnel, error) {
	if tunnelID == "" {
		return nil, validationError("TunnelService.Get", "tunnel ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetTunnel(ctx, rc, tunnelID)
	if err != nil {
		return nil, newError("TunnelService.Get", fmt.Sprintf("failed to get tunnel %q", tunnelID), err)
	}

	return mapTunnel(result), nil
}

// Resolve returns the tunnel identified by an ID or a name. Names are matched
// against the account's tunnels; zero or multiple matches are errors.
func (s *TunnelService) Resolve(ctx context.Context, idOrName string) (*Tunnel, error) {
	if idOrName == "" {
		return nil, validationError("TunnelService.Resolve", "tunnel ID or name is required")
	}

	if tunnelIDPattern.MatchString(idOrName) {
		return s.Get(ctx, idOrName)
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListTunnels(ctx, rc, cloudflare.TunnelListParams{Name: idOrName})
	if err != nil {
		return nil, newError("TunnelService.Resolve", fmt.Sprintf("failed to look up tunnel by name %q", idOrName), err)
	}

	matches := make([]*Tunnel, 0, len(results))
	for _, t := range results {
		if t.Name == idOrName {
			matches = append(matches, mapTunnel(t))
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nil, newError("TunnelService.Resolve",
			fmt.Sprintf("no tunnel named %q found in the account — check the name or use the tunnel ID from 'cosmoflare tunnel list'", idOrName), nil)
	default:
		return nil, newError("TunnelService.Resolve",
			fmt.Sprintf("name %q matches %d tunnels — use a tunnel ID instead", idOrName, len(matches)), nil)
	}
}

// Create creates a new tunnel. configSrc selects the configuration source:
// "cloudflare" for remotely-managed tunnels, empty for locally-managed ones.
// The API requires a client-supplied tunnel secret, so a random one is
// generated and sent with the create call; the secret is intentionally not
// part of the returned Tunnel — connectors authenticate with the token from
// Token() instead.
func (s *TunnelService) Create(ctx context.Context, name, configSrc string) (*Tunnel, error) {
	if name == "" {
		return nil, validationError("TunnelService.Create", "tunnel name is required")
	}

	secret, err := tunnelGenerateSecret()
	if err != nil {
		return nil, newError("TunnelService.Create", "failed to generate tunnel secret", err)
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.TunnelCreateParams{
		Name:      name,
		ConfigSrc: configSrc,
		Secret:    secret,
	}

	result, err := s.cf.CreateTunnel(ctx, rc, params)
	if err != nil {
		return nil, newError("TunnelService.Create", fmt.Sprintf("failed to create tunnel %q", name), err)
	}

	return mapTunnel(result), nil
}

// Delete deletes a tunnel. When cascade is true, connector connections are
// torn down first so the delete does not race active connectors (the SDK's
// DeleteTunnel does not expose the API's ?cascade=true query, so the teardown
// is performed as an explicit preceding call).
func (s *TunnelService) Delete(ctx context.Context, tunnelID string, cascade bool) error {
	if tunnelID == "" {
		return validationError("TunnelService.Delete", "tunnel ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)

	if cascade {
		if err := s.cf.CleanupTunnelConnections(ctx, rc, tunnelID); err != nil {
			return newError("TunnelService.Delete", fmt.Sprintf("failed to clean up connections for tunnel %q before delete", tunnelID), err)
		}
	}

	if err := s.cf.DeleteTunnel(ctx, rc, tunnelID); err != nil {
		return newError("TunnelService.Delete", fmt.Sprintf("failed to delete tunnel %q", tunnelID), err)
	}
	return nil
}

// Token returns the connector token for a tunnel, suitable for
// `cloudflared tunnel run --token <token>`.
func (s *TunnelService) Token(ctx context.Context, tunnelID string) (string, error) {
	if tunnelID == "" {
		return "", validationError("TunnelService.Token", "tunnel ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	token, err := s.cf.GetTunnelToken(ctx, rc, tunnelID)
	if err != nil {
		return "", newError("TunnelService.Token", fmt.Sprintf("failed to get token for tunnel %q", tunnelID), err)
	}
	return token, nil
}

// Connections lists the connectors currently attached to a tunnel.
func (s *TunnelService) Connections(ctx context.Context, tunnelID string) ([]*TunnelConnector, error) {
	if tunnelID == "" {
		return nil, validationError("TunnelService.Connections", "tunnel ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	results, err := s.cf.ListTunnelConnections(ctx, rc, tunnelID)
	if err != nil {
		return nil, newError("TunnelService.Connections", fmt.Sprintf("failed to list connections for tunnel %q", tunnelID), err)
	}

	connectors := make([]*TunnelConnector, 0, len(results))
	for _, c := range results {
		connectors = append(connectors, mapTunnelConnector(c))
	}
	return connectors, nil
}

// CleanupTunnelConnections force-disconnects the connectors attached to a
// tunnel without deleting the tunnel itself.
func (s *TunnelService) CleanupTunnelConnections(ctx context.Context, tunnelID string) error {
	if tunnelID == "" {
		return validationError("TunnelService.CleanupTunnelConnections", "tunnel ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.CleanupTunnelConnections(ctx, rc, tunnelID); err != nil {
		return newError("TunnelService.CleanupTunnelConnections", fmt.Sprintf("failed to clean up connections for tunnel %q", tunnelID), err)
	}
	return nil
}

// Cleanup tears down a tunnel completely: connections first, then the tunnel.
func (s *TunnelService) Cleanup(ctx context.Context, tunnelID string) error {
	if tunnelID == "" {
		return validationError("TunnelService.Cleanup", "tunnel ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.CleanupTunnelConnections(ctx, rc, tunnelID); err != nil {
		return newError("TunnelService.Cleanup", fmt.Sprintf("failed to clean up connections for tunnel %q", tunnelID), err)
	}
	if err := s.cf.DeleteTunnel(ctx, rc, tunnelID); err != nil {
		return newError("TunnelService.Cleanup", fmt.Sprintf("failed to delete tunnel %q", tunnelID), err)
	}
	return nil
}

// tunnelGenerateSecret produces the base64-encoded 32-byte tunnel secret the
// Cloudflare API requires on create.
func tunnelGenerateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// mapTunnel converts a cloudflare.Tunnel to our Tunnel type.
func mapTunnel(t cloudflare.Tunnel) *Tunnel {
	return &Tunnel{
		ID:           t.ID,
		Name:         t.Name,
		CreatedAt:    t.CreatedAt,
		DeletedAt:    t.DeletedAt,
		Status:       t.Status,
		TunnelType:   t.TunnelType,
		RemoteConfig: t.RemoteConfig,
		Connections:  mapTunnelConnections(t.Connections),
	}
}

// mapTunnelConnector converts a cloudflare.Connection to our TunnelConnector type.
func mapTunnelConnector(c cloudflare.Connection) *TunnelConnector {
	return &TunnelConnector{
		ID:          c.ID,
		Features:    c.Features,
		Version:     c.Version,
		Arch:        c.Arch,
		RunAt:       c.RunAt,
		Connections: mapTunnelConnections(c.Connections),
	}
}

// mapTunnelConnections converts SDK tunnel connections to our type.
func mapTunnelConnections(conns []cloudflare.TunnelConnection) []TunnelConnection {
	if len(conns) == 0 {
		return nil
	}
	mapped := make([]TunnelConnection, 0, len(conns))
	for _, c := range conns {
		mapped = append(mapped, TunnelConnection{
			ID:                 c.ID,
			ColoName:           c.ColoName,
			ClientID:           c.ClientID,
			ClientVersion:      c.ClientVersion,
			OpenedAt:           c.OpenedAt,
			OriginIP:           c.OriginIP,
			IsPendingReconnect: c.IsPendingReconnect,
		})
	}
	return mapped
}
