// Package cmdmanifest is the per-command metadata spine (FEAT-020): one
// registry describing every CLI command — CLI path, wrangler equivalent,
// service, verb, API operations, required token permissions, limit catalog
// ids, preflight checks, rate-limit behavior, danger level, and tracking
// flags. Declared once here, consumed everywhere: audit logging stamps
// danger levels, dry-run defaults derive from the destructive flag,
// permission checks resolve scopes per command, and MCP tool generation
// derives descriptions.
//
// The registry is Go source on purpose: compile-time typed, diffable in
// review, and impossible to ship half-loaded. Permissions columns are
// intentionally sparse until the FEAT-011 catalog dataset lands; an empty
// list means "not yet authored", never "none required".
package cmdmanifest

// Command describes one CLI command. Field semantics follow the corpus
// schema (docs/research/2026-09-10-cf-limits-corpus, section 4.1).
type Command struct {
	ID                 string   // dotted unique id, e.g. "r2.object.put"
	CLIPath            []string // invocation path, e.g. ["r2", "object", "put"]
	WranglerEquivalent string   // closest wrangler command, "" when none exists
	Service            string   // owning service tag (r2, workers, kv, dns...)
	Scope              string   // resource granularity: account|zone|bucket|object|...
	Verb               string   // read | write | delete | list
	APIOps             []string // endpoints the command touches (method + path)
	Permissions        Permissions
	Limits             []string // limit catalog ids guarding this command
	LocalChecks        []string // preflight checks run before any API call
	APIChecks          []string // checks run against the API before mutating
	RateLimit          *RateLimitBehavior
	DangerLevel        string // low | medium | high
	Destructive        bool
	Trackable          bool
}

// Permissions lists the API-token permission names a command requires, per
// scope. Names follow the live token-UI spelling (e.g. "Workers R2 Storage").
type Permissions struct {
	Account []string
	Zone    []string
	User    []string
}

// RateLimitBehavior records how the backing API throttles this command.
type RateLimitBehavior struct {
	HTTPStatus int  // status signaling the limit (429)
	RetryAfter bool // whether Retry-After is honored on that status
	Bucket     string // rate bucket name, e.g. "r2_rest_or_s3"
}

// Manifest is the loaded registry with lookup indexes.
type Manifest struct {
	commands []Command
	byID     map[string]int
	byPath   map[string]int
}

// Load returns the built-in manifest. The data is immutable; the returned
// Manifest may be shared freely.
func Load() *Manifest {
	m := &Manifest{
		commands: registry,
		byID:     make(map[string]int, len(registry)),
		byPath:   make(map[string]int, len(registry)),
	}
	for i, c := range registry {
		m.byID[c.ID] = i
		m.byPath[pathKey(c.CLIPath)] = i
	}
	return m
}

// Commands returns the registry in declaration order.
func (m *Manifest) Commands() []Command { return m.commands }

// Get returns the command with the given dotted id.
func (m *Manifest) Get(id string) (Command, bool) {
	i, ok := m.byID[id]
	if !ok {
		return Command{}, false
	}
	return m.commands[i], true
}

// ResolveCLI returns the command invoked by the given path segments.
func (m *Manifest) ResolveCLI(path ...string) (Command, bool) {
	if len(path) == 0 {
		return Command{}, false
	}
	i, ok := m.byPath[pathKey(path)]
	if !ok {
		return Command{}, false
	}
	return m.commands[i], true
}

func pathKey(path []string) string {
	key := ""
	for i, seg := range path {
		if i > 0 {
			key += " "
		}
		key += seg
	}
	return key
}
