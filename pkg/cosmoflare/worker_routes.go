package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// WorkerRoute maps traffic matching a URL pattern to a Worker script,
// scoped to a Cloudflare zone.
type WorkerRoute struct {
	ID      string `json:"id"`
	Pattern string `json:"pattern"`
	Script  string `json:"script"`
}

// RouteList returns all Worker routes in the zone.
func (s *WorkerService) RouteList(ctx context.Context, zoneID string) ([]WorkerRoute, error) {
	if zoneID == "" {
		return nil, validationError("WorkerService.RouteList", "zone ID is required")
	}

	rc := cloudflare.ZoneIdentifier(zoneID)
	resp, err := s.cf.ListWorkerRoutes(ctx, rc, cloudflare.ListWorkerRoutesParams{})
	if err != nil {
		return nil, newError("WorkerService.RouteList", fmt.Sprintf("failed to list worker routes for zone %q", zoneID), err)
	}

	routes := make([]WorkerRoute, 0, len(resp.Routes))
	for _, r := range resp.Routes {
		routes = append(routes, WorkerRoute{
			ID:      r.ID,
			Pattern: r.Pattern,
			Script:  r.ScriptName,
		})
	}
	return routes, nil
}

// RouteCreate maps a URL pattern to a Worker script in the zone.
func (s *WorkerService) RouteCreate(ctx context.Context, zoneID, pattern, script string) (*WorkerRoute, error) {
	if zoneID == "" {
		return nil, validationError("WorkerService.RouteCreate", "zone ID is required")
	}
	if pattern == "" {
		return nil, validationError("WorkerService.RouteCreate", "pattern is required")
	}
	if script == "" {
		return nil, validationError("WorkerService.RouteCreate", "script is required")
	}

	rc := cloudflare.ZoneIdentifier(zoneID)
	resp, err := s.cf.CreateWorkerRoute(ctx, rc, cloudflare.CreateWorkerRouteParams{
		Pattern: pattern,
		Script:  script,
	})
	if err != nil {
		return nil, newError("WorkerService.RouteCreate", fmt.Sprintf("failed to create worker route %q", pattern), err)
	}

	return &WorkerRoute{
		ID:      resp.ID,
		Pattern: resp.Pattern,
		Script:  resp.ScriptName,
	}, nil
}

// RouteUpdate changes the pattern and/or script of an existing Worker route.
func (s *WorkerService) RouteUpdate(ctx context.Context, zoneID, routeID, pattern, script string) (*WorkerRoute, error) {
	if zoneID == "" {
		return nil, validationError("WorkerService.RouteUpdate", "zone ID is required")
	}
	if routeID == "" {
		return nil, validationError("WorkerService.RouteUpdate", "route ID is required")
	}
	if pattern == "" {
		return nil, validationError("WorkerService.RouteUpdate", "pattern is required")
	}
	if script == "" {
		return nil, validationError("WorkerService.RouteUpdate", "script is required")
	}

	rc := cloudflare.ZoneIdentifier(zoneID)
	resp, err := s.cf.UpdateWorkerRoute(ctx, rc, cloudflare.UpdateWorkerRouteParams{
		ID:      routeID,
		Pattern: pattern,
		Script:  script,
	})
	if err != nil {
		return nil, newError("WorkerService.RouteUpdate", fmt.Sprintf("failed to update worker route %q", routeID), err)
	}

	return &WorkerRoute{
		ID:      resp.ID,
		Pattern: resp.Pattern,
		Script:  resp.ScriptName,
	}, nil
}

// RouteDelete removes a Worker route from the zone.
func (s *WorkerService) RouteDelete(ctx context.Context, zoneID, routeID string) error {
	if zoneID == "" {
		return validationError("WorkerService.RouteDelete", "zone ID is required")
	}
	if routeID == "" {
		return validationError("WorkerService.RouteDelete", "route ID is required")
	}

	rc := cloudflare.ZoneIdentifier(zoneID)
	_, err := s.cf.DeleteWorkerRoute(ctx, rc, routeID)
	if err != nil {
		return newError("WorkerService.RouteDelete", fmt.Sprintf("failed to delete worker route %q", routeID), err)
	}
	return nil
}
