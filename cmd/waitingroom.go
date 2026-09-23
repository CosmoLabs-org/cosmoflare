package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Waiting Room is zone-scoped; a zone ID is required for all operations.
// Required token permission: zone "Waiting Room".

var waitingRoomCmd = &cobra.Command{
	Use:   "waiting-room",
	Short: "Manage Cloudflare Waiting Rooms",
	Long: `Waiting Room management for Cloudflare zones.

Waiting Rooms hold excess visitors in a customizable queue instead of
letting an overwhelmed origin serve degraded responses (or drop requests
entirely).

Commands:
  create    Create a waiting room
  list      List waiting rooms
  get       Get a waiting room
  update    Update a waiting room
  delete    Delete a waiting room

Waiting Rooms are zone-scoped, so a zone ID is required for all operations.

Examples:
  cosmoflare waiting-room create ZONE_ID --name=launch --host=example.com --new-users-per-minute=200 --total-active-users=500
  cosmoflare waiting-room list ZONE_ID --json
  cosmoflare waiting-room get ZONE_ID ROOM_ID
  cosmoflare waiting-room update ZONE_ID ROOM_ID --total-active-users=750
  cosmoflare waiting-room delete ZONE_ID ROOM_ID --force`,
}

var (
	wrName              string
	wrHost              string
	wrPath              string
	wrDescription       string
	wrQueueingMethod    string
	wrNewUsersPerMinute int
	wrTotalActiveUsers  int
	wrSessionDuration   int
	wrQueueAll          bool
	wrDisableRenewal    bool
	wrSuspended         bool
	wrJSONResponse      bool
	wrQueueingStatus    int
	wrCustomPageHTML    string
	wrForce             bool
)

var waitingRoomCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a waiting room",
	Long: `Create a new waiting room in a Cloudflare zone.

Required: --name, --host, --new-users-per-minute, --total-active-users.

Sensible defaults (matching the Cloudflare API defaults) when flags are
omitted: --path=/ (room covers the whole host), --session-duration=1
(minutes), --queueing-status-code=429, FIFO queueing method.

Examples:
  cosmoflare waiting-room create ZONE_ID --name=launch --host=example.com --new-users-per-minute=200 --total-active-users=500
  cosmoflare waiting-room create ZONE_ID --name=sale --host=shop.example.com --path=/checkout --new-users-per-minute=100 --total-active-users=300 --queueing-method=random
  cosmoflare waiting-room create ZONE_ID --name=launch --host=example.com --new-users-per-minute=200 --total-active-users=500 --session-duration=5 --queue-all --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(1)),
	RunE: runWaitingRoomCreate,
}

var waitingRoomListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List waiting rooms",
	Long: `List all waiting rooms in a zone.

Examples:
  cosmoflare waiting-room list ZONE_ID
  cosmoflare waiting-room list ZONE_ID --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(1)),
	RunE: runWaitingRoomList,
}

var waitingRoomGetCmd = &cobra.Command{
	Use:   "get [zone-id] [room-id]",
	Short: "Get a waiting room",
	Long: `Get details of a single waiting room by its ID.

Examples:
  cosmoflare waiting-room get ZONE_ID ROOM_ID
  cosmoflare waiting-room get ZONE_ID ROOM_ID --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(2)),
	RunE: runWaitingRoomGet,
}

var waitingRoomUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [room-id]",
	Short: "Update a waiting room",
	Long: `Update an existing waiting room.

Provide only the fields you want to change. Unspecified fields keep their
current values.

Examples:
  cosmoflare waiting-room update ZONE_ID ROOM_ID --total-active-users=750
  cosmoflare waiting-room update ZONE_ID ROOM_ID --new-users-per-minute=400 --session-duration=10
  cosmoflare waiting-room update ZONE_ID ROOM_ID --suspend
  cosmoflare waiting-room update ZONE_ID ROOM_ID --queue-all --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(2)),
	RunE: runWaitingRoomUpdate,
}

var waitingRoomDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [room-id]",
	Short: "Delete a waiting room",
	Long: `Delete a waiting room from a zone.

WARNING: This action is irreversible. All queued visitors are released
immediately and the room's configuration is lost.

Examples:
  cosmoflare waiting-room delete ZONE_ID ROOM_ID
  cosmoflare waiting-room delete ZONE_ID ROOM_ID --force`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(2)),
	RunE: runWaitingRoomDelete,
}

func init() {
	rootCmd.AddCommand(waitingRoomCmd)

	waitingRoomCmd.AddCommand(waitingRoomCreateCmd)
	waitingRoomCmd.AddCommand(waitingRoomListCmd)
	waitingRoomCmd.AddCommand(waitingRoomGetCmd)
	waitingRoomCmd.AddCommand(waitingRoomUpdateCmd)
	waitingRoomCmd.AddCommand(waitingRoomDeleteCmd)

	// Create flags. Defaults mirror the documented Cloudflare API defaults.
	waitingRoomCreateCmd.Flags().StringVar(&wrName, "name", "", "Waiting room name")
	waitingRoomCreateCmd.Flags().StringVar(&wrHost, "host", "", "Host the waiting room applies to (e.g., example.com)")
	waitingRoomCreateCmd.Flags().StringVar(&wrPath, "path", "/", "Path the waiting room covers (default: whole host)")
	waitingRoomCreateCmd.Flags().IntVar(&wrNewUsersPerMinute, "new-users-per-minute", 0, "New users let out of the queue per minute (required)")
	waitingRoomCreateCmd.Flags().IntVar(&wrTotalActiveUsers, "total-active-users", 0, "Maximum concurrent active users on the origin (required)")
	waitingRoomCreateCmd.Flags().IntVar(&wrSessionDuration, "session-duration", 1, "Session duration in minutes (default 1)")
	waitingRoomCreateCmd.Flags().StringVar(&wrDescription, "description", "", "Description of the waiting room")
	waitingRoomCreateCmd.Flags().StringVar(&wrQueueingMethod, "queueing-method", "fifo", "Queueing method: fifo, random, passthrough")
	waitingRoomCreateCmd.Flags().BoolVar(&wrQueueAll, "queue-all", false, "Queue all incoming users immediately")
	waitingRoomCreateCmd.Flags().BoolVar(&wrDisableRenewal, "disable-session-renewal", false, "Disable session renewal")
	waitingRoomCreateCmd.Flags().BoolVar(&wrSuspended, "suspended", false, "Create the room suspended")
	waitingRoomCreateCmd.Flags().BoolVar(&wrJSONResponse, "json-response", false, "Return wait-time JSON instead of HTML")
	waitingRoomCreateCmd.Flags().IntVar(&wrQueueingStatus, "queueing-status-code", 429, "HTTP status code served while queueing (default 429)")
	waitingRoomCreateCmd.Flags().StringVar(&wrCustomPageHTML, "custom-page-html", "", "Custom waiting-room HTML")
	_ = waitingRoomCreateCmd.MarkFlagRequired("name")
	_ = waitingRoomCreateCmd.MarkFlagRequired("host")
	_ = waitingRoomCreateCmd.MarkFlagRequired("new-users-per-minute")
	_ = waitingRoomCreateCmd.MarkFlagRequired("total-active-users")

	// Update flags: every field is optional; at least one must change.
	waitingRoomUpdateCmd.Flags().StringVar(&wrName, "name", "", "New waiting room name")
	waitingRoomUpdateCmd.Flags().StringVar(&wrHost, "host", "", "New host")
	waitingRoomUpdateCmd.Flags().StringVar(&wrPath, "path", "", "New path")
	waitingRoomUpdateCmd.Flags().IntVar(&wrNewUsersPerMinute, "new-users-per-minute", 0, "New users per minute")
	waitingRoomUpdateCmd.Flags().IntVar(&wrTotalActiveUsers, "total-active-users", 0, "Maximum concurrent active users")
	waitingRoomUpdateCmd.Flags().IntVar(&wrSessionDuration, "session-duration", 0, "Session duration in minutes")
	waitingRoomUpdateCmd.Flags().StringVar(&wrDescription, "description", "", "New description")
	waitingRoomUpdateCmd.Flags().StringVar(&wrQueueingMethod, "queueing-method", "", "Queueing method: fifo, random, passthrough")
	waitingRoomUpdateCmd.Flags().BoolVar(&wrQueueAll, "queue-all", false, "Queue all incoming users immediately")
	waitingRoomUpdateCmd.Flags().BoolVar(&wrSuspended, "suspend", false, "Suspend the waiting room")
	waitingRoomUpdateCmd.Flags().BoolVar(&wrDisableRenewal, "disable-session-renewal", false, "Disable session renewal")
	waitingRoomUpdateCmd.Flags().BoolVar(&wrJSONResponse, "json-response", false, "Return wait-time JSON instead of HTML")
	waitingRoomUpdateCmd.Flags().IntVar(&wrQueueingStatus, "queueing-status-code", 0, "HTTP status code served while queueing")
	waitingRoomUpdateCmd.Flags().StringVar(&wrCustomPageHTML, "custom-page-html", "", "Custom waiting-room HTML")

	// Delete flags
	waitingRoomDeleteCmd.Flags().BoolVar(&wrForce, "force", false, "Skip confirmation prompt")
}

func getWaitingRoomService(zoneID string) (*cosmoflare.WaitingRoomService, error) {
	return cosmoflare.NewWaitingRoomServiceFromCreds(zoneID, APIToken)
}

func runWaitingRoomCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getWaitingRoomService(zoneID)
	if err != nil {
		return outErr("failed to create Waiting Room service", err)
	}

	opts := cosmoflare.WaitingRoomCreate{
		Name:                  wrName,
		Host:                  wrHost,
		Path:                  wrPath,
		Description:           wrDescription,
		QueueingMethod:        wrQueueingMethod,
		NewUsersPerMinute:     wrNewUsersPerMinute,
		TotalActiveUsers:      wrTotalActiveUsers,
		SessionDuration:       wrSessionDuration,
		QueueAll:              wrQueueAll,
		DisableSessionRenewal: wrDisableRenewal,
		Suspended:             wrSuspended,
		JsonResponseEnabled:   wrJSONResponse,
		QueueingStatusCode:    wrQueueingStatus,
		CustomPageHTML:        wrCustomPageHTML,
	}

	if DryRun {
		return outPayload("DRY RUN: Would create waiting room", func() any {
			return map[string]interface{}{
				"zone_id":              zoneID,
				"name":                 wrName,
				"host":                 wrHost,
				"path":                 wrPath,
				"new_users_per_minute": wrNewUsersPerMinute,
				"total_active_users":   wrTotalActiveUsers,
			}
		}, func() {
			printInfo("DRY RUN: Would create waiting room '%s' on %s%s in zone %s", wrName, wrHost, wrPath, zoneID)
		})
	}

	room, err := svc.Create(context.Background(), opts)
	if err != nil {
		return outErr("failed to create waiting room", err)
	}

	return outPayload("Waiting room created successfully", func() any {
		return room
	}, func() {
		printSuccess("Waiting room created successfully!")
		printInfo("ID: %s", room.ID)
		printInfo("Name: %s", room.Name)
		printInfo("Host: %s%s", room.Host, room.Path)
		printInfo("New users/minute: %d", room.NewUsersPerMinute)
		printInfo("Total active users: %d", room.TotalActiveUsers)
	})
}

func runWaitingRoomList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getWaitingRoomService(zoneID)
	if err != nil {
		return outErr("failed to create Waiting Room service", err)
	}

	rooms, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list waiting rooms", err)
	}

	return outResult(rooms, func() {
		if len(rooms) == 0 {
			printInfo("No waiting rooms found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tHOST\tPATH\tQUEUEING\tUSERS/MIN\tACTIVE")
		for _, r := range rooms {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
				r.ID, r.Name, r.Host, r.Path, r.QueueingMethod, r.NewUsersPerMinute, r.TotalActiveUsers)
		}
		w.Flush()

		printInfo("Total: %d waiting room(s)", len(rooms))
	})
}

func runWaitingRoomGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and room ID are required")
	}
	zoneID, roomID := args[0], args[1]

	svc, err := getWaitingRoomService(zoneID)
	if err != nil {
		return outErr("failed to create Waiting Room service", err)
	}

	room, err := svc.Get(context.Background(), roomID)
	if err != nil {
		return outErr("failed to get waiting room", err)
	}

	return outResult(room, func() {
		fmt.Printf("ID:                  %s\n", room.ID)
		fmt.Printf("Name:                %s\n", room.Name)
		fmt.Printf("Host:                %s%s\n", room.Host, room.Path)
		if room.Description != "" {
			fmt.Printf("Description:         %s\n", room.Description)
		}
		fmt.Printf("Queueing method:     %s\n", room.QueueingMethod)
		fmt.Printf("New users/minute:    %d\n", room.NewUsersPerMinute)
		fmt.Printf("Total active users:  %d\n", room.TotalActiveUsers)
		fmt.Printf("Session duration:    %d min\n", room.SessionDuration)
		fmt.Printf("Queue all:           %v\n", room.QueueAll)
		fmt.Printf("Suspended:           %v\n", room.Suspended)
		fmt.Printf("Queueing status:     %d\n", room.QueueingStatusCode)
		if room.CustomPageHTML != "" {
			fmt.Printf("Custom page HTML:    %d bytes\n", len(room.CustomPageHTML))
		}
		fmt.Printf("Created:             %s\n", room.CreatedOn)
		fmt.Printf("Modified:            %s\n", room.ModifiedOn)
	})
}

func runWaitingRoomUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and room ID are required")
	}
	zoneID, roomID := args[0], args[1]

	var opts cosmoflare.WaitingRoomUpdate
	changed := 0
	if cmd.Flags().Changed("name") {
		opts.Name = &wrName
		changed++
	}
	if cmd.Flags().Changed("host") {
		opts.Host = &wrHost
		changed++
	}
	if cmd.Flags().Changed("path") {
		opts.Path = &wrPath
		changed++
	}
	if cmd.Flags().Changed("description") {
		opts.Description = &wrDescription
		changed++
	}
	if cmd.Flags().Changed("queueing-method") {
		opts.QueueingMethod = &wrQueueingMethod
		changed++
	}
	if cmd.Flags().Changed("new-users-per-minute") {
		opts.NewUsersPerMinute = &wrNewUsersPerMinute
		changed++
	}
	if cmd.Flags().Changed("total-active-users") {
		opts.TotalActiveUsers = &wrTotalActiveUsers
		changed++
	}
	if cmd.Flags().Changed("session-duration") {
		opts.SessionDuration = &wrSessionDuration
		changed++
	}
	if cmd.Flags().Changed("queue-all") {
		opts.QueueAll = &wrQueueAll
		changed++
	}
	if cmd.Flags().Changed("disable-session-renewal") {
		opts.DisableSessionRenewal = &wrDisableRenewal
		changed++
	}
	if cmd.Flags().Changed("suspend") {
		opts.Suspended = &wrSuspended
		changed++
	}
	if cmd.Flags().Changed("json-response") {
		opts.JsonResponseEnabled = &wrJSONResponse
		changed++
	}
	if cmd.Flags().Changed("queueing-status-code") {
		opts.QueueingStatusCode = &wrQueueingStatus
		changed++
	}
	if cmd.Flags().Changed("custom-page-html") {
		opts.CustomPageHTML = &wrCustomPageHTML
		changed++
	}
	if changed == 0 {
		return fmt.Errorf("at least one update flag is required (--name, --host, --path, --description, --queueing-method, --new-users-per-minute, --total-active-users, --session-duration, --queue-all, --disable-session-renewal, --suspend, --json-response, --queueing-status-code, --custom-page-html)")
	}

	svc, err := getWaitingRoomService(zoneID)
	if err != nil {
		return outErr("failed to create Waiting Room service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update waiting room", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"room_id": roomID,
			}
		}, func() {
			printInfo("DRY RUN: Would update waiting room '%s' in zone '%s'", roomID, zoneID)
		})
	}

	room, err := svc.Update(context.Background(), roomID, opts)
	if err != nil {
		return outErr("failed to update waiting room", err)
	}

	return outPayload("Waiting room updated successfully", func() any {
		return room
	}, func() {
		printSuccess("Waiting room '%s' updated successfully!", roomID)
		printInfo("Name: %s  Host: %s%s  Users/min: %d  Active: %d",
			room.Name, room.Host, room.Path, room.NewUsersPerMinute, room.TotalActiveUsers)
	})
}

func runWaitingRoomDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and room ID are required")
	}
	zoneID, roomID := args[0], args[1]

	// Not registry-flagged destructive yet (registry pass follows): the
	// confirmation prompt stays live; an explicit --dry-run still
	// short-circuits before any service call.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "waiting-room delete"), wrForce)
	if dry {
		return outPayload("DRY RUN: Would delete waiting room", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"room_id": roomID,
			}
		}, func() {
			printInfo("DRY RUN: Would delete waiting room '%s' from zone '%s'", roomID, zoneID)
		})
	}
	if !wrForce && !ux.Confirm(fmt.Sprintf("Delete waiting room '%s'? All queued visitors are released immediately", roomID)) {
		printInfo("Waiting room deletion cancelled")
		return nil
	}

	svc, err := getWaitingRoomService(zoneID)
	if err != nil {
		return outErr("failed to create Waiting Room service", err)
	}

	if err := svc.Delete(context.Background(), roomID); err != nil {
		return outErr("failed to delete waiting room", err)
	}

	return outPayload("Waiting room deleted successfully", func() any {
		return map[string]string{
			"zone_id": zoneID,
			"room_id": roomID,
		}
	}, func() {
		printSuccess("Waiting room '%s' deleted successfully!", roomID)
	})
}
