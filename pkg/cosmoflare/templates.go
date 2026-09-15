package cosmoflare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Template describes a built-in project scaffold.
type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Services    []string `json:"services"`
	Files       []string `json:"files"`
}

// TemplateFile holds a file path and its content for scaffolding.
type TemplateFile struct {
	Path    string
	Content string
}

// CreateResult holds the outcome of a template scaffold operation.
type CreateResult struct {
	TemplateName string   `json:"template_name"`
	ProjectName  string   `json:"project_name"`
	Directory    string   `json:"directory"`
	FilesCreated []string `json:"files_created"`
}

// TemplateService provides operations for listing and scaffolding Cloudflare
// project templates. All templates are embedded — no network required.
type TemplateService struct {
	templates map[string]Template
}

// NewTemplateService creates a new TemplateService with the built-in templates.
func NewTemplateService() *TemplateService {
	svc := &TemplateService{
		templates: make(map[string]Template),
	}
	svc.registerBuiltins()
	return svc
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// ListTemplates returns all available templates sorted by name.
func (s *TemplateService) ListTemplates() []Template {
	result := make([]Template, 0, len(s.templates))
	for _, t := range s.templates {
		result = append(result, t)
	}
	// Stable sort by name
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Name > result[j].Name {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// GetTemplate returns a single template by name.
func (s *TemplateService) GetTemplate(name string) (*Template, error) {
	if name == "" {
		return nil, validationError("TemplateService.GetTemplate", "template name is required")
	}
	t, ok := s.templates[name]
	if !ok {
		available := make([]string, 0, len(s.templates))
		for k := range s.templates {
			available = append(available, k)
		}
		return nil, validationError("TemplateService.GetTemplate",
			fmt.Sprintf("unknown template %q (available: %s)", name, strings.Join(available, ", ")))
	}
	return &t, nil
}

// CreateFromTemplate scaffolds a project from a named template into dir.
// If force is false, it refuses to overwrite existing files.
func (s *TemplateService) CreateFromTemplate(name, dir, projectName string, force bool) (*CreateResult, error) {
	if dir == "" {
		return nil, validationError("TemplateService.CreateFromTemplate", "directory is required")
	}
	if name == "" {
		return nil, validationError("TemplateService.CreateFromTemplate", "template name is required")
	}

	tmpl, ok := s.templates[name]
	if !ok {
		return nil, validationError("TemplateService.CreateFromTemplate",
			fmt.Sprintf("unknown template %q", name))
	}

	if projectName == "" {
		projectName = filepath.Base(dir)
	}

	files := s.filesForTemplate(name, projectName)

	// Pre-check for existing files when force=false
	if !force {
		for _, f := range files {
			fullPath := filepath.Join(dir, f.Path)
			if _, err := os.Stat(fullPath); err == nil {
				return nil, fmt.Errorf("file %q already exists (use --force to overwrite)", f.Path)
			}
		}
	}

	var created []string
	for _, f := range files {
		fullPath := filepath.Join(dir, f.Path)

		// Create parent directories
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %q: %w", parentDir, err)
		}

		if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %q: %w", f.Path, err)
		}
		created = append(created, f.Path)
	}

	return &CreateResult{
		TemplateName: tmpl.Name,
		ProjectName:  projectName,
		Directory:    dir,
		FilesCreated: created,
	}, nil
}

// ---------------------------------------------------------------------------
// Built-in template registry
// ---------------------------------------------------------------------------

func (s *TemplateService) registerBuiltins() {
	s.templates["api"] = Template{
		Name:        "api",
		Description: "Worker API with KV bindings, CORS, and structured error handling",
		Services:    []string{"Workers", "KV"},
		Files:       []string{".cosmoflare.yaml", "wrangler.toml", "src/index.js", "src/router.js"},
	}
	s.templates["static"] = Template{
		Name:        "static",
		Description: "Pages static site with build pipeline and deployment config",
		Services:    []string{"Pages"},
		Files:       []string{".cosmoflare.yaml", "wrangler.toml", "public/index.html", "public/style.css"},
	}
	s.templates["fullstack"] = Template{
		Name:        "fullstack",
		Description: "Worker API + Pages frontend + R2 storage + D1 database",
		Services:    []string{"Workers", "Pages", "R2", "D1"},
		Files:       []string{".cosmoflare.yaml", "wrangler.toml", "src/api/index.js", "src/api/db.js", "src/frontend/index.html", "src/frontend/app.js"},
	}
	s.templates["cron"] = Template{
		Name:        "cron",
		Description: "Scheduled Worker with KV state persistence",
		Services:    []string{"Workers", "KV"},
		Files:       []string{".cosmoflare.yaml", "wrangler.toml", "src/index.js", "src/state.js"},
	}
	s.templates["queue"] = Template{
		Name:        "queue",
		Description: "Queue producer + consumer Workers with dead-letter handling",
		Services:    []string{"Workers", "Queues"},
		Files:       []string{".cosmoflare.yaml", "wrangler.toml", "src/producer.js", "src/consumer.js"},
	}
}

// ---------------------------------------------------------------------------
// Template file content generators
// ---------------------------------------------------------------------------

func (s *TemplateService) filesForTemplate(name, projectName string) []TemplateFile {
	switch name {
	case "api":
		return s.apiFiles(projectName)
	case "static":
		return s.staticFiles(projectName)
	case "fullstack":
		return s.fullstackFiles(projectName)
	case "cron":
		return s.cronFiles(projectName)
	case "queue":
		return s.queueFiles(projectName)
	default:
		return nil
	}
}

func (s *TemplateService) apiFiles(name string) []TemplateFile {
	return []TemplateFile{
		{Path: ".cosmoflare.yaml", Content: s.apiConfigYAML(name)},
		{Path: "wrangler.toml", Content: s.apiWranglerToml(name)},
		{Path: "src/index.js", Content: s.apiIndexJS()},
		{Path: "src/router.js", Content: s.apiRouterJS()},
	}
}

func (s *TemplateService) apiConfigYAML(name string) string {
	return fmt.Sprintf(`# Cosmoflare Project Configuration
# Template: api — Worker API with KV bindings

name: %q
type: worker

workers:
  api:
    script: src/index.js
    compatibility_date: "2024-01-01"
    module: true
    kv_namespaces:
      - binding: DATA
        id: ""

routes:
  - pattern: "api.example.com/*"
    zone_name: "example.com"
`, name)
}

func (s *TemplateService) apiWranglerToml(name string) string {
	return fmt.Sprintf(`name = %q
main = "src/index.js"
compatibility_date = "2024-01-01"

[vars]
ENVIRONMENT = "production"

[[kv_namespaces]]
binding = "DATA"
id = ""
`, name)
}

func (s *TemplateService) apiIndexJS() string {
	return `// Worker API entry point
// Generated by: cosmoflare templates create api

import { handleRequest } from './router.js';

export default {
  async fetch(request, env, ctx) {
    try {
      return await handleRequest(request, env);
    } catch (err) {
      return new Response(JSON.stringify({ error: err.message }), {
        status: 500,
        headers: { 'Content-Type': 'application/json' },
      });
    }
  },
};
`
}

func (s *TemplateService) apiRouterJS() string {
	return `// Simple request router with CORS support
// Generated by: cosmoflare templates create api

const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization',
};

export async function handleRequest(request, env) {
  if (request.method === 'OPTIONS') {
    return new Response(null, { headers: corsHeaders });
  }

  const url = new URL(request.url);

  switch (url.pathname) {
    case '/':
      return json({ status: 'ok', service: env.ENVIRONMENT || 'development' });
    case '/health':
      return json({ healthy: true });
    default:
      return json({ error: 'Not Found' }, 404);
  }
}

function json(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json', ...corsHeaders },
  });
}
`
}

func (s *TemplateService) staticFiles(name string) []TemplateFile {
	return []TemplateFile{
		{
			Path: ".cosmoflare.yaml",
			Content: fmt.Sprintf(`# Cosmoflare Project Configuration
# Template: static — Pages static site

name: %q
type: pages

pages:
  build_command: ""
  build_output: public
  production_branch: main
`, name),
		},
		{
			Path: "wrangler.toml",
			Content: fmt.Sprintf(`name = %q
pages_build_output_dir = "public"

# Uncomment for custom build:
# [build]
# command = "npm run build"
`, name),
		},
		{
			Path: "public/index.html",
			Content: fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <main>
    <h1>%s</h1>
    <p>Deployed with Cloudflare Pages via Cosmoflare.</p>
  </main>
</body>
</html>
`, name, name),
		},
		{
			Path: "public/style.css",
			Content: `/* Generated by: cosmoflare templates create static */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, sans-serif; line-height: 1.6; max-width: 40rem; margin: 2rem auto; padding: 0 1rem; }
h1 { margin-bottom: 0.5rem; }
`,
		},
	}
}

func (s *TemplateService) fullstackFiles(name string) []TemplateFile {
	return []TemplateFile{
		{Path: ".cosmoflare.yaml", Content: s.fullstackConfigYAML(name)},
		{Path: "wrangler.toml", Content: s.fullstackWranglerToml(name)},
		{Path: "src/api/index.js", Content: s.fullstackAPIIndexJS()},
		{Path: "src/api/db.js", Content: s.fullstackDBJS()},
		{Path: "src/frontend/index.html", Content: s.fullstackIndexHTML(name)},
		{Path: "src/frontend/app.js", Content: s.fullstackAppJS()},
	}
}

func (s *TemplateService) fullstackConfigYAML(name string) string {
	return fmt.Sprintf(`# Cosmoflare Project Configuration
# Template: fullstack — Worker API + Pages frontend + R2 + D1

name: %q
type: full

workers:
  api:
    script: src/api/index.js
    compatibility_date: "2024-01-01"
    module: true
    kv_namespaces: []
    r2_buckets:
      - binding: ASSETS
        bucket_name: "%s-assets"
    d1_databases:
      - binding: DB
        database_name: "%s-db"
        database_id: ""

pages:
  build_command: ""
  build_output: src/frontend

r2:
  buckets:
    - name: "%s-assets"
      location: auto
`, name, name, name, name)
}

func (s *TemplateService) fullstackWranglerToml(name string) string {
	return fmt.Sprintf(`name = %q
main = "src/api/index.js"
compatibility_date = "2024-01-01"

[[r2_buckets]]
binding = "ASSETS"
bucket_name = "%s-assets"

[[d1_databases]]
binding = "DB"
database_name = "%s-db"
database_id = ""
`, name, name, name)
}

func (s *TemplateService) fullstackAPIIndexJS() string {
	return `// Fullstack API entry point
// Generated by: cosmoflare templates create fullstack

import { initDB } from './db.js';

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // API routes
    if (url.pathname.startsWith('/api/')) {
      return handleAPI(request, env);
    }

    // Serve static frontend for everything else
    return new Response('See /api/ for the API', { status: 200 });
  },
};

async function handleAPI(request, env) {
  const url = new URL(request.url);

  switch (url.pathname) {
    case '/api/health':
      return Response.json({ healthy: true });
    case '/api/init':
      await initDB(env.DB);
      return Response.json({ message: 'Database initialized' });
    default:
      return Response.json({ error: 'Not Found' }, { status: 404 });
  }
}
`
}

func (s *TemplateService) fullstackDBJS() string {
	return "// D1 database helper\n" +
		"// Generated by: cosmoflare templates create fullstack\n\n" +
		"export async function initDB(db) {\n" +
		"  await db.exec(`\n" +
		"    CREATE TABLE IF NOT EXISTS items (\n" +
		"      id INTEGER PRIMARY KEY AUTOINCREMENT,\n" +
		"      name TEXT NOT NULL,\n" +
		"      created_at DATETIME DEFAULT CURRENT_TIMESTAMP\n" +
		"    )\n" +
		"  `);\n" +
		"}\n\n" +
		"export async function listItems(db) {\n" +
		"  const { results } = await db.prepare('SELECT * FROM items ORDER BY created_at DESC').all();\n" +
		"  return results;\n" +
		"}\n\n" +
		"export async function createItem(db, name) {\n" +
		"  const result = await db.prepare('INSERT INTO items (name) VALUES (?)').bind(name).run();\n" +
		"  return result;\n" +
		"}\n"
}

func (s *TemplateService) fullstackIndexHTML(name string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body>
  <div id="app"></div>
  <script src="app.js"></script>
</body>
</html>
`, name)
}

func (s *TemplateService) fullstackAppJS() string {
	return `// Frontend application
// Generated by: cosmoflare templates create fullstack

async function init() {
  const res = await fetch('/api/health');
  const data = await res.json();
  document.getElementById('app').innerHTML =
    '<h1>Fullstack App</h1><p>API status: ' + (data.healthy ? 'healthy' : 'down') + '</p>';
}

init();
`
}

func (s *TemplateService) cronFiles(name string) []TemplateFile {
	return []TemplateFile{
		{Path: ".cosmoflare.yaml", Content: s.cronConfigYAML(name)},
		{Path: "wrangler.toml", Content: s.cronWranglerToml(name)},
		{Path: "src/index.js", Content: s.cronIndexJS()},
		{Path: "src/state.js", Content: s.cronStateJS()},
	}
}

func (s *TemplateService) cronConfigYAML(name string) string {
	return fmt.Sprintf(`# Cosmoflare Project Configuration
# Template: cron — Scheduled Worker with KV state

name: %q
type: worker

workers:
  cron:
    script: src/index.js
    compatibility_date: "2024-01-01"
    module: true
    kv_namespaces:
      - binding: STATE
        id: ""
    triggers:
      crons:
        - "0 * * * *"
`, name)
}

func (s *TemplateService) cronWranglerToml(name string) string {
	return fmt.Sprintf(`name = %q
main = "src/index.js"
compatibility_date = "2024-01-01"

[[kv_namespaces]]
binding = "STATE"
id = ""

[triggers]
crons = ["0 * * * *"]
`, name)
}

func (s *TemplateService) cronIndexJS() string {
	return `// Scheduled Worker entry point
// Generated by: cosmoflare templates create cron

import { loadState, saveState } from './state.js';

export default {
  async scheduled(event, env, ctx) {
    const state = await loadState(env.STATE);
    console.log('Cron triggered at', new Date().toISOString(), 'run #' + (state.runCount + 1));

    state.runCount += 1;
    state.lastRun = new Date().toISOString();

    // TODO: Add your scheduled logic here

    await saveState(env.STATE, state);
  },

  async fetch(request, env) {
    const state = await loadState(env.STATE);
    return Response.json({
      status: 'ok',
      lastRun: state.lastRun,
      runCount: state.runCount,
    });
  },
};
`
}

func (s *TemplateService) cronStateJS() string {
	return `// KV-backed state persistence
// Generated by: cosmoflare templates create cron

const STATE_KEY = 'cron-state';

const defaultState = {
  runCount: 0,
  lastRun: null,
};

export async function loadState(kv) {
  const raw = await kv.get(STATE_KEY, { type: 'json' });
  return raw || { ...defaultState };
}

export async function saveState(kv, state) {
  await kv.put(STATE_KEY, JSON.stringify(state));
}
`
}

func (s *TemplateService) queueFiles(name string) []TemplateFile {
	return []TemplateFile{
		{Path: ".cosmoflare.yaml", Content: s.queueConfigYAML(name)},
		{Path: "wrangler.toml", Content: s.queueWranglerToml(name)},
		{Path: "src/producer.js", Content: s.queueProducerJS()},
		{Path: "src/consumer.js", Content: s.queueConsumerJS()},
	}
}

func (s *TemplateService) queueConfigYAML(name string) string {
	return fmt.Sprintf(`# Cosmoflare Project Configuration
# Template: queue — Producer + Consumer Workers

name: %q
type: worker

workers:
  producer:
    script: src/producer.js
    compatibility_date: "2024-01-01"
    module: true
    queues:
      producers:
        - queue: "%s-queue"
          binding: QUEUE
  consumer:
    script: src/consumer.js
    compatibility_date: "2024-01-01"
    module: true
    queues:
      consumers:
        - queue: "%s-queue"
          max_batch_size: 10
          max_retries: 3
          dead_letter_queue: "%s-dlq"
`, name, name, name, name)
}

func (s *TemplateService) queueWranglerToml(name string) string {
	return fmt.Sprintf(`name = %q
main = "src/producer.js"
compatibility_date = "2024-01-01"

[[queues.producers]]
queue = "%s-queue"
binding = "QUEUE"

[[queues.consumers]]
queue = "%s-queue"
max_batch_size = 10
max_retries = 3
dead_letter_queue = "%s-dlq"
`, name, name, name, name)
}

func (s *TemplateService) queueProducerJS() string {
	return `// Queue producer Worker
// Generated by: cosmoflare templates create queue

export default {
  async fetch(request, env) {
    if (request.method !== 'POST') {
      return Response.json({ error: 'POST required' }, { status: 405 });
    }

    try {
      const body = await request.json();
      await env.QUEUE.send({
        type: body.type || 'default',
        payload: body.payload || {},
        timestamp: new Date().toISOString(),
      });

      return Response.json({ queued: true });
    } catch (err) {
      return Response.json({ error: err.message }, { status: 500 });
    }
  },
};
`
}

func (s *TemplateService) queueConsumerJS() string {
	return `// Queue consumer Worker
// Generated by: cosmoflare templates create queue

export default {
  async queue(batch, env) {
    for (const message of batch.messages) {
      try {
        console.log('Processing message:', JSON.stringify(message.body));

        // TODO: Add your message processing logic here
        switch (message.body.type) {
          case 'default':
            // Handle default message type
            break;
          default:
            console.warn('Unknown message type:', message.body.type);
        }

        message.ack();
      } catch (err) {
        console.error('Failed to process message:', err.message);
        message.retry();
      }
    }
  },
};
`
}
