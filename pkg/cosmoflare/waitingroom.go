package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// WaitingRoomInfo is the public view of a zone-scoped Cloudflare Waiting
// Room. Timestamps are rendered as RFC 3339 strings, empty when the API
// reports none. Waiting Rooms hold excess visitors in a customizable queue
// when a origin would otherwise be overwhelmed.
//
// Permissions{Zone: []string{"Waiting Room"}} — required token permission
// for this service (registry pass consumes this note).
type WaitingRoomInfo struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Host                  string `json:"host"`
	Path                  string `json:"path"`
	Description           string `json:"description,omitempty"`
	QueueingMethod        string `json:"queueing_method,omitempty"`
	NewUsersPerMinute     int    `json:"new_users_per_minute"`
	TotalActiveUsers      int    `json:"total_active_users"`
	SessionDuration       int    `json:"session_duration"`
	QueueAll              bool   `json:"queue_all"`
	DisableSessionRenewal bool   `json:"disable_session_renewal"`
	Suspended             bool   `json:"suspended"`
	JsonResponseEnabled   bool   `json:"json_response_enabled"`
	QueueingStatusCode    int    `json:"queueing_status_code"`
	CustomPageHTML        string `json:"custom_page_html,omitempty"`
	CookieSuffix          string `json:"cookie_suffix,omitempty"`
	CreatedOn             string `json:"created_on,omitempty"`
	ModifiedOn            string `json:"modified_on,omitempty"`
}

// WaitingRoomCreate carries the fields for waiting-room creation. Name,
// Host, NewUsersPerMinute and TotalActiveUsers are required by the API;
// everything else falls back to the documented Cloudflare defaults
// (path "/", session_duration 1, queueing_status_code 429, FIFO queueing).
type WaitingRoomCreate struct {
	Name                  string
	Host                  string
	Path                  string
	Description           string
	QueueingMethod        string
	NewUsersPerMinute     int
	TotalActiveUsers      int
	SessionDuration       int
	QueueAll              bool
	DisableSessionRenewal bool
	Suspended             bool
	JsonResponseEnabled   bool
	QueueingStatusCode    int
	CustomPageHTML        string
}

// WaitingRoomUpdate carries optional changes to an existing waiting room.
// nil fields are left untouched; the service merges the patch onto the
// room's current state before issuing the PATCH so untouched API-required
// fields keep their live values.
type WaitingRoomUpdate struct {
	Name                  *string
	Host                  *string
	Path                  *string
	Description           *string
	QueueingMethod        *string
	NewUsersPerMinute     *int
	TotalActiveUsers      *int
	SessionDuration       *int
	QueueAll              *bool
	DisableSessionRenewal *bool
	Suspended             *bool
	JsonResponseEnabled   *bool
	QueueingStatusCode    *int
	CustomPageHTML        *string
}

// WaitingRoomService implements zone-scoped Waiting Room operations on top
// of the typed cloudflare-go resources.
//
// Permissions{Zone: []string{"Waiting Room"}}
type WaitingRoomService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewWaitingRoomService creates a new Waiting Room service client.
func NewWaitingRoomService(api *cloudflare.API, zoneID string) (*WaitingRoomService, error) {
	if api == nil {
		return nil, validationError("NewWaitingRoomService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewWaitingRoomService", "zone ID is required")
	}
	return &WaitingRoomService{cf: api, zoneID: zoneID}, nil
}

// NewWaitingRoomServiceFromCreds creates a WaitingRoomService from zone ID
// and API token. Convenience helper for CLI usage.
func NewWaitingRoomServiceFromCreds(zoneID, apiToken string) (*WaitingRoomService, error) {
	if zoneID == "" {
		return nil, validationError("NewWaitingRoomServiceFromCreds", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewWaitingRoomServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewWaitingRoomServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &WaitingRoomService{cf: cf, zoneID: zoneID}, nil
}

// Create adds a new waiting room to the zone. The returned room carries the
// ID Cloudflare assigned.
func (s *WaitingRoomService) Create(ctx context.Context, opts WaitingRoomCreate) (*WaitingRoomInfo, error) {
	if opts.Name == "" {
		return nil, validationError("WaitingRoomService.Create", "name is required")
	}
	if opts.Host == "" {
		return nil, validationError("WaitingRoomService.Create", "host is required")
	}
	if opts.NewUsersPerMinute <= 0 {
		return nil, validationError("WaitingRoomService.Create", "new users per minute must be positive")
	}
	if opts.TotalActiveUsers <= 0 {
		return nil, validationError("WaitingRoomService.Create", "total active users must be positive")
	}

	room, err := s.cf.CreateWaitingRoom(ctx, s.zoneID, cloudflare.WaitingRoom{
		Name:                  opts.Name,
		Host:                  opts.Host,
		Path:                  opts.Path,
		Description:           opts.Description,
		QueueingMethod:        opts.QueueingMethod,
		NewUsersPerMinute:     opts.NewUsersPerMinute,
		TotalActiveUsers:      opts.TotalActiveUsers,
		SessionDuration:       opts.SessionDuration,
		QueueAll:              opts.QueueAll,
		DisableSessionRenewal: opts.DisableSessionRenewal,
		Suspended:             opts.Suspended,
		JsonResponseEnabled:   opts.JsonResponseEnabled,
		QueueingStatusCode:    opts.QueueingStatusCode,
		CustomPageHTML:        opts.CustomPageHTML,
	})
	if err != nil {
		return nil, newError("WaitingRoomService.Create", fmt.Sprintf("failed to create waiting room %q", opts.Name), err)
	}
	mapped := waitingRoomFromCF(*room)
	return &mapped, nil
}

// List returns every waiting room in the zone.
func (s *WaitingRoomService) List(ctx context.Context) ([]WaitingRoomInfo, error) {
	rooms, err := s.cf.ListWaitingRooms(ctx, s.zoneID)
	if err != nil {
		return nil, newError("WaitingRoomService.List", fmt.Sprintf("failed to list waiting rooms in zone %q", s.zoneID), err)
	}
	out := make([]WaitingRoomInfo, 0, len(rooms))
	for i := range rooms {
		out = append(out, waitingRoomFromCF(rooms[i]))
	}
	return out, nil
}

// Get fetches a single waiting room by ID.
func (s *WaitingRoomService) Get(ctx context.Context, roomID string) (*WaitingRoomInfo, error) {
	if roomID == "" {
		return nil, validationError("WaitingRoomService.Get", "waiting room ID is required")
	}

	room, err := s.cf.WaitingRoom(ctx, s.zoneID, roomID)
	if err != nil {
		return nil, newError("WaitingRoomService.Get", fmt.Sprintf("failed to get waiting room %q", roomID), err)
	}
	mapped := waitingRoomFromCF(room)
	return &mapped, nil
}

// Update applies a partial change to a waiting room. The room's current
// state is fetched first and the requested fields merged onto it, so the
// PATCH body keeps untouched required fields at their live values.
func (s *WaitingRoomService) Update(ctx context.Context, roomID string, opts WaitingRoomUpdate) (*WaitingRoomInfo, error) {
	if roomID == "" {
		return nil, validationError("WaitingRoomService.Update", "waiting room ID is required")
	}

	current, err := s.cf.WaitingRoom(ctx, s.zoneID, roomID)
	if err != nil {
		return nil, newError("WaitingRoomService.Update", fmt.Sprintf("failed to fetch waiting room %q before update", roomID), err)
	}

	patched := current
	if opts.Name != nil {
		patched.Name = *opts.Name
	}
	if opts.Host != nil {
		patched.Host = *opts.Host
	}
	if opts.Path != nil {
		patched.Path = *opts.Path
	}
	if opts.Description != nil {
		patched.Description = *opts.Description
	}
	if opts.QueueingMethod != nil {
		patched.QueueingMethod = *opts.QueueingMethod
	}
	if opts.NewUsersPerMinute != nil {
		patched.NewUsersPerMinute = *opts.NewUsersPerMinute
	}
	if opts.TotalActiveUsers != nil {
		patched.TotalActiveUsers = *opts.TotalActiveUsers
	}
	if opts.SessionDuration != nil {
		patched.SessionDuration = *opts.SessionDuration
	}
	if opts.QueueAll != nil {
		patched.QueueAll = *opts.QueueAll
	}
	if opts.DisableSessionRenewal != nil {
		patched.DisableSessionRenewal = *opts.DisableSessionRenewal
	}
	if opts.Suspended != nil {
		patched.Suspended = *opts.Suspended
	}
	if opts.JsonResponseEnabled != nil {
		patched.JsonResponseEnabled = *opts.JsonResponseEnabled
	}
	if opts.QueueingStatusCode != nil {
		patched.QueueingStatusCode = *opts.QueueingStatusCode
	}
	if opts.CustomPageHTML != nil {
		patched.CustomPageHTML = *opts.CustomPageHTML
	}

	room, err := s.cf.ChangeWaitingRoom(ctx, s.zoneID, roomID, patched)
	if err != nil {
		return nil, newError("WaitingRoomService.Update", fmt.Sprintf("failed to update waiting room %q", roomID), err)
	}
	mapped := waitingRoomFromCF(room)
	return &mapped, nil
}

// Delete removes a waiting room from the zone.
func (s *WaitingRoomService) Delete(ctx context.Context, roomID string) error {
	if roomID == "" {
		return validationError("WaitingRoomService.Delete", "waiting room ID is required")
	}

	if err := s.cf.DeleteWaitingRoom(ctx, s.zoneID, roomID); err != nil {
		return newError("WaitingRoomService.Delete", fmt.Sprintf("failed to delete waiting room %q", roomID), err)
	}
	return nil
}

// waitingRoomFromCF maps the cloudflare-go waiting room to the public
// WaitingRoomInfo view, rendering timestamps as RFC3339 strings and
// omitting zero times entirely.
func waitingRoomFromCF(r cloudflare.WaitingRoom) WaitingRoomInfo {
	created, modified := "", ""
	if !r.CreatedOn.IsZero() {
		created = r.CreatedOn.Format(time.RFC3339)
	}
	if !r.ModifiedOn.IsZero() {
		modified = r.ModifiedOn.Format(time.RFC3339)
	}
	return WaitingRoomInfo{
		ID:                    r.ID,
		Name:                  r.Name,
		Host:                  r.Host,
		Path:                  r.Path,
		Description:           r.Description,
		QueueingMethod:        r.QueueingMethod,
		NewUsersPerMinute:     r.NewUsersPerMinute,
		TotalActiveUsers:      r.TotalActiveUsers,
		SessionDuration:       r.SessionDuration,
		QueueAll:              r.QueueAll,
		DisableSessionRenewal: r.DisableSessionRenewal,
		Suspended:             r.Suspended,
		JsonResponseEnabled:   r.JsonResponseEnabled,
		QueueingStatusCode:    r.QueueingStatusCode,
		CustomPageHTML:        r.CustomPageHTML,
		CookieSuffix:          r.CookieSuffix,
		CreatedOn:             created,
		ModifiedOn:            modified,
	}
}
