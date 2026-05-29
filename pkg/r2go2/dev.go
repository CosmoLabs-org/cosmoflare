package r2go2

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// DevServer is a local development proxy for Cloudflare services.
type DevServer struct {
	port     int
	watch    bool
	services []string
	profile  string

	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
	ready    bool
	stopped  bool
}

// DevOption configures a DevServer.
type DevOption func(*DevServer)

// WithDevPort sets the listening port. 0 means pick a free port.
func WithDevPort(port int) DevOption {
	return func(ds *DevServer) { ds.port = port }
}

// WithDevWatch enables or disables config file watching.
func WithDevWatch(w bool) DevOption {
	return func(ds *DevServer) { ds.watch = w }
}

// WithDevServices restricts which service routes are enabled.
func WithDevServices(services []string) DevOption {
	return func(ds *DevServer) { ds.services = services }
}

// WithDevProfile sets the credential profile name.
func WithDevProfile(profile string) DevOption {
	return func(ds *DevServer) { ds.profile = profile }
}

// allServices is the default set when no filter is applied.
var allServices = []string{"r2", "kv", "workers", "dns", "zones", "ssl", "cache", "d1", "pages", "queues"}

// NewDevServer creates a DevServer with the given options.
func NewDevServer(opts ...DevOption) *DevServer {
	ds := &DevServer{
		port:     8787,
		watch:    true,
		services: nil,
	}
	for _, o := range opts {
		o(ds)
	}
	if ds.services == nil {
		ds.services = allServices
	}
	return ds
}

// Port returns the configured port.
func (ds *DevServer) Port() int { return ds.port }

// Watch returns whether config watching is enabled.
func (ds *DevServer) Watch() bool { return ds.watch }

// Services returns the active service list.
func (ds *DevServer) Services() []string { return ds.services }

// Profile returns the credential profile name.
func (ds *DevServer) Profile() string { return ds.profile }

// Ready returns true once the server is listening.
func (ds *DevServer) Ready() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.ready
}

// Addr returns the listener address (e.g. "127.0.0.1:8787").
// Empty string if not started.
func (ds *DevServer) Addr() string {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.listener == nil {
		return ""
	}
	return ds.listener.Addr().String()
}

// DevStartupEvent holds machine-readable startup information.
type DevStartupEvent struct {
	Port     int      `json:"port"`
	Address  string   `json:"address"`
	Services []string `json:"services"`
	Watch    bool     `json:"watch"`
	Profile  string   `json:"profile,omitempty"`
}

// StartupEvent returns the JSON-serializable startup information.
func (ds *DevServer) StartupEvent() DevStartupEvent {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	addr := ""
	port := ds.port
	if ds.listener != nil {
		addr = ds.listener.Addr().String()
		port = ds.listener.Addr().(*net.TCPAddr).Port
	}
	return DevStartupEvent{
		Port:     port,
		Address:  addr,
		Services: ds.services,
		Watch:    ds.watch,
		Profile:  ds.profile,
	}
}

// Start begins serving and blocks until ctx is cancelled.
// Returns nil on clean shutdown.
func (ds *DevServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	serviceSet := make(map[string]bool, len(ds.services))
	for _, s := range ds.services {
		serviceSet[s] = true
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	for _, svc := range ds.services {
		svcName := svc
		mux.HandleFunc("/"+svcName+"/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, `{"error":"upstream not configured","service":"%s"}`, svcName)
		})
		mux.HandleFunc("/"+svcName, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, `{"error":"upstream not configured","service":"%s"}`, svcName)
		})
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"service":"cosmoflare-dev"}`)
			return
		}
		// Check if the first path segment matches a configured service
		path := r.URL.Path[1:] // strip leading /
		for i := 0; i < len(path); i++ {
			if path[i] == '/' {
				path = path[:i]
				break
			}
		}
		if !serviceSet[path] {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"service not found","path":"%s"}`, r.URL.Path)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error":"upstream not configured","service":"%s"}`, path)
	})

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ds.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", ds.port, err)
	}

	ds.mu.Lock()
	ds.listener = ln
	ds.port = ln.Addr().(*net.TCPAddr).Port
	ds.server = &http.Server{Handler: mux}
	ds.ready = true
	ds.mu.Unlock()

	go func() {
		<-ctx.Done()
		ds.Stop()
	}()

	err = ds.server.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Stop gracefully shuts down the server.
func (ds *DevServer) Stop() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.stopped || ds.server == nil {
		return
	}
	ds.stopped = true
	ds.ready = false
	_ = ds.server.Close()
}
