package cmdmanifest

// registry is the command registry in declaration order. Wave 1 authors the
// r2 pilot group (FEAT-020); later waves append groups without touching the
// loader. Permissions come from the live token-UI names verified 2026-09-18
// (docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md);
// entries left empty await the FEAT-011 dataset, never "none required".
var registry = []Command{
	// --- r2 bucket lifecycle ---
	{
		ID: "r2.bucket.create", CLIPath: []string{"r2", "bucket", "create"},
		WranglerEquivalent: "wrangler r2 bucket create",
		Service:            "r2", Scope: "account", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/r2/buckets"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		Limits:      []string{"r2.buckets"},
		LocalChecks: []string{"name_charset"},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.bucket.list", CLIPath: []string{"r2", "bucket", "list"},
		WranglerEquivalent: "wrangler r2 bucket list",
		Service:            "r2", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/r2/buckets"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.bucket.get", CLIPath: []string{"r2", "bucket", "get"},
		WranglerEquivalent: "wrangler r2 bucket info",
		Service:            "r2", Scope: "bucket", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/r2/buckets/{bucket}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.bucket.exists", CLIPath: []string{"r2", "bucket", "exists"},
		Service: "r2", Scope: "bucket", Verb: "read",
		APIOps:      []string{"HEAD /accounts/{account_id}/r2/buckets/{bucket}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.bucket.delete", CLIPath: []string{"r2", "bucket", "delete"},
		WranglerEquivalent: "wrangler r2 bucket delete",
		Service:            "r2", Scope: "bucket", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/r2/buckets/{bucket}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		APIChecks:   []string{"bucket_empty"},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest"},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},

	// --- r2 object operations ---
	{
		ID: "r2.object.put", CLIPath: []string{"r2", "object", "put"},
		WranglerEquivalent: "wrangler r2 object put",
		Service:            "r2", Scope: "object", Verb: "write",
		APIOps: []string{
			"PUT /{bucket}/{key}",
			"POST multipart-upload", "POST upload-part", "POST complete-multipart-upload",
		},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		Limits:      []string{"r2.upload_single", "r2.upload_multipart", "r2.object_metadata"},
		LocalChecks: []string{"file_size", "metadata_size", "key_length"},
		APIChecks:   []string{"bucket_exists", "api_rate_remaining"},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.object.get", CLIPath: []string{"r2", "object", "get"},
		WranglerEquivalent: "wrangler r2 object get",
		Service:            "r2", Scope: "object", Verb: "read",
		APIOps:      []string{"GET /{bucket}/{key}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.object.delete", CLIPath: []string{"r2", "object", "delete"},
		WranglerEquivalent: "wrangler r2 object delete",
		Service:            "r2", Scope: "object", Verb: "delete",
		APIOps:      []string{"DELETE /{bucket}/{key}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.object.list", CLIPath: []string{"r2", "object", "list"},
		Service: "r2", Scope: "bucket", Verb: "list",
		APIOps:      []string{"GET /{bucket}?list-type=2"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// --- r2 sync / watch (workflow commands over the object surface) ---
	{
		ID: "sync.up", CLIPath: []string{"sync", "up"},
		Service: "r2", Scope: "bucket", Verb: "write",
		APIOps:      []string{"PUT /{bucket}/{key}", "POST multipart-upload"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		Limits:      []string{"r2.upload_single", "r2.upload_multipart"},
		LocalChecks: []string{"directory_exists", "exclude_patterns"},
		APIChecks:   []string{"bucket_exists"},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "sync.down", CLIPath: []string{"sync", "down"},
		Service: "r2", Scope: "bucket", Verb: "read",
		APIOps:      []string{"GET /{bucket}?list-type=2", "GET /{bucket}/{key}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		LocalChecks: []string{"destination_writable"},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "watch", CLIPath: []string{"watch"},
		Service: "r2", Scope: "bucket", Verb: "write",
		APIOps:      []string{"PUT /{bucket}/{key}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		LocalChecks: []string{"directory_exists"},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- worker script lifecycle (wave 2; permissions from the Qwen dataset) ---
	{
		ID: "worker.deploy", CLIPath: []string{"worker", "deploy"},
		WranglerEquivalent: "wrangler deploy",
		Service:            "workers", Scope: "script", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/scripts/{script_name}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.list", CLIPath: []string{"worker", "list"},
		Service: "workers", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.get", CLIPath: []string{"worker", "get"},
		Service: "workers", Scope: "script", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.delete", CLIPath: []string{"worker", "delete"},
		WranglerEquivalent: "wrangler delete",
		Service:            "workers", Scope: "script", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/workers/scripts/{script_name}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		APIChecks:   []string{"script_exists"},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},
	{
		ID: "worker.logs", CLIPath: []string{"worker", "logs"},
		Service: "workers", Scope: "script", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/observability/telemetry/query"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.tail", CLIPath: []string{"worker", "tail"},
		WranglerEquivalent: "wrangler tail",
		Service:            "workers", Scope: "script", Verb: "read",
		APIOps:      []string{"POST /accounts/{account_id}/workers/scripts/{script_name}/tails"},
		Permissions: Permissions{Account: []string{"Workers Tail", "Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.settings", CLIPath: []string{"worker", "settings"},
		Service: "workers", Scope: "script", Verb: "write",
		APIOps:      []string{"PATCH /accounts/{account_id}/workers/scripts/{script_name}/settings"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.bindings", CLIPath: []string{"worker", "bindings"},
		Service: "workers", Scope: "script", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/settings"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.types", CLIPath: []string{"worker", "types"},
		WranglerEquivalent: "wrangler types",
		Service:            "workers", Scope: "script", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/settings"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		LocalChecks: []string{"bindings_present"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.rollback", CLIPath: []string{"worker", "rollback"},
		WranglerEquivalent: "wrangler rollback",
		Service:            "workers", Scope: "script", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/workers/scripts/{script_name}/versions/{version_id}/deploy"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- worker cron triggers (schedules are a full-set replace server-side) ---
	{
		ID: "worker.cron.list", CLIPath: []string{"worker", "cron", "list"},
		Service: "workers", Scope: "script", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/schedules"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.cron.create", CLIPath: []string{"worker", "cron", "create"},
		Service: "workers", Scope: "script", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/scripts/{script_name}/schedules"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.cron.update", CLIPath: []string{"worker", "cron", "update"},
		Service: "workers", Scope: "script", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/scripts/{script_name}/schedules"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.cron.delete", CLIPath: []string{"worker", "cron", "delete"},
		Service: "workers", Scope: "script", Verb: "delete",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/scripts/{script_name}/schedules"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- worker secrets (values are write-only; only names ever surface) ---
	{
		ID: "worker.secret.put", CLIPath: []string{"worker", "secret", "put"},
		WranglerEquivalent: "wrangler secret put",
		Service:            "workers", Scope: "secret", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/scripts/{script_name}/secrets"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.secret.bulk", CLIPath: []string{"worker", "secret", "bulk"},
		WranglerEquivalent: "wrangler secret bulk",
		Service:            "workers", Scope: "secret", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/scripts/{script_name}/secrets/bulk"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		LocalChecks: []string{"secrets_file_json"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.secret.list", CLIPath: []string{"worker", "secret", "list"},
		WranglerEquivalent: "wrangler secret list",
		Service:            "workers", Scope: "secret", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/secrets"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.secret.delete", CLIPath: []string{"worker", "secret", "delete"},
		WranglerEquivalent: "wrangler secret delete",
		Service:            "workers", Scope: "secret", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/workers/scripts/{script_name}/secrets/{secret_name}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- worker domains (permission group pending the FEAT-011 dataset) ---
	{
		ID: "worker.domain.list", CLIPath: []string{"worker", "domain", "list"},
		Service: "workers", Scope: "domain", Verb: "list",
		APIOps: []string{"GET /accounts/{account_id}/workers/domains"},
		// "Workers Custom Domains" not yet in the Qwen dataset — awaiting
		// FEAT-011 ingest rather than guessing the token-UI spelling.
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.domain.attach", CLIPath: []string{"worker", "domain", "attach"},
		Service: "workers", Scope: "domain", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/workers/domains"},
		LocalChecks: []string{"dns_record_resolves"},
		// permission pending FEAT-011 dataset
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.domain.detach", CLIPath: []string{"worker", "domain", "detach"},
		Service: "workers", Scope: "domain", Verb: "delete",
		APIOps: []string{"DELETE /accounts/{account_id}/workers/domains/{domain_id}"},
		// permission pending FEAT-011 dataset
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- worker routes (zone-scoped; zone permission from the Qwen dataset) ---
	{
		ID: "worker.route.list", CLIPath: []string{"worker", "route", "list"},
		Service: "workers", Scope: "route", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/workers/routes"},
		Permissions: Permissions{Zone: []string{"Workers Routes"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.route.create", CLIPath: []string{"worker", "route", "create"},
		Service: "workers", Scope: "route", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/workers/routes"},
		Permissions: Permissions{Zone: []string{"Workers Routes"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.route.update", CLIPath: []string{"worker", "route", "update"},
		Service: "workers", Scope: "route", Verb: "write",
		APIOps:      []string{"PUT /zones/{zone_id}/workers/routes/{route_id}"},
		Permissions: Permissions{Zone: []string{"Workers Routes"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.route.delete", CLIPath: []string{"worker", "route", "delete"},
		Service: "workers", Scope: "route", Verb: "delete",
		APIOps:      []string{"DELETE /zones/{zone_id}/workers/routes/{route_id}"},
		Permissions: Permissions{Zone: []string{"Workers Routes"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- workers.dev subdomain ---
	{
		ID: "worker.subdomain.get", CLIPath: []string{"worker", "subdomain", "get"},
		Service: "workers", Scope: "account", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/subdomain"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.subdomain.set", CLIPath: []string{"worker", "subdomain", "set"},
		Service: "workers", Scope: "account", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/workers/subdomain"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		LocalChecks: []string{"subdomain_charset"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- worker versions & deployments ---
	{
		ID: "worker.versions.list", CLIPath: []string{"worker", "versions", "list"},
		WranglerEquivalent: "wrangler versions list",
		Service:            "workers", Scope: "version", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/versions"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.versions.upload", CLIPath: []string{"worker", "versions", "upload"},
		WranglerEquivalent: "wrangler versions upload",
		Service:            "workers", Scope: "version", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/workers/scripts/{script_name}/versions"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.versions.view", CLIPath: []string{"worker", "versions", "view"},
		WranglerEquivalent: "wrangler versions view",
		Service:            "workers", Scope: "version", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/versions/{version_id}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.versions.deploy", CLIPath: []string{"worker", "versions", "deploy"},
		WranglerEquivalent: "wrangler versions deploy",
		Service:            "workers", Scope: "version", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/workers/scripts/{script_name}/versions/{version_id}/deploy"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.versions.rollback", CLIPath: []string{"worker", "versions", "rollback"},
		WranglerEquivalent: "wrangler rollback",
		Service:            "workers", Scope: "version", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/workers/scripts/{script_name}/versions/{version_id}/deploy"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.versions.delete", CLIPath: []string{"worker", "versions", "delete"},
		Service: "workers", Scope: "version", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/workers/scripts/{script_name}/versions/{version_id}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.deployments.list", CLIPath: []string{"worker", "deployments", "list"},
		WranglerEquivalent: "wrangler deployments list",
		Service:            "workers", Scope: "deployment", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/deployments"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "worker.deployments.view", CLIPath: []string{"worker", "deployments", "view"},
		WranglerEquivalent: "wrangler deployments view",
		Service:            "workers", Scope: "deployment", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/workers/scripts/{script_name}/deployments/{deployment_id}"},
		Permissions: Permissions{Account: []string{"Workers Scripts"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// --- kv namespaces ---
	{
		ID: "kv.namespace.create", CLIPath: []string{"kv", "namespace", "create"},
		WranglerEquivalent: "wrangler kv namespace create",
		Service:            "kv", Scope: "namespace", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/storage/kv/namespaces"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "kv.namespace.list", CLIPath: []string{"kv", "namespace", "list"},
		WranglerEquivalent: "wrangler kv namespace list",
		Service:            "kv", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/storage/kv/namespaces"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "kv.namespace.delete", CLIPath: []string{"kv", "namespace", "delete"},
		WranglerEquivalent: "wrangler kv namespace delete",
		Service:            "kv", Scope: "namespace", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/storage/kv/namespaces/{namespace_id}"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		APIChecks:   []string{"namespace_empty"},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},

	// --- kv keys ---
	{
		ID: "kv.put", CLIPath: []string{"kv", "put"},
		WranglerEquivalent: "wrangler kv key put",
		Service:            "kv", Scope: "key", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/storage/kv/namespaces/{namespace_id}/values/{key}"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		LocalChecks: []string{"key_charset"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "kv.get", CLIPath: []string{"kv", "get"},
		WranglerEquivalent: "wrangler kv key get",
		Service:            "kv", Scope: "key", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/storage/kv/namespaces/{namespace_id}/values/{key}"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "kv.delete", CLIPath: []string{"kv", "delete"},
		WranglerEquivalent: "wrangler kv key delete",
		Service:            "kv", Scope: "key", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/storage/kv/namespaces/{namespace_id}/values/{key}"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "kv.list", CLIPath: []string{"kv", "list"},
		WranglerEquivalent: "wrangler kv key list",
		Service:            "kv", Scope: "namespace", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/storage/kv/namespaces/{namespace_id}/keys"},
		Permissions: Permissions{Account: []string{"Workers KV Storage"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// --- d1 databases ---
	{
		ID: "d1.create", CLIPath: []string{"d1", "create"},
		WranglerEquivalent: "wrangler d1 create",
		Service:            "d1", Scope: "database", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database"},
		Permissions: Permissions{Account: []string{"D1"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.list", CLIPath: []string{"d1", "list"},
		WranglerEquivalent: "wrangler d1 list",
		Service:            "d1", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/d1/database"},
		Permissions: Permissions{Account: []string{"D1"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.get", CLIPath: []string{"d1", "get"},
		WranglerEquivalent: "wrangler d1 info",
		Service:            "d1", Scope: "database", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/d1/database/{database_id}"},
		Permissions: Permissions{Account: []string{"D1"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.delete", CLIPath: []string{"d1", "delete"},
		WranglerEquivalent: "wrangler d1 delete",
		Service:            "d1", Scope: "database", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/d1/database/{database_id}"},
		Permissions: Permissions{Account: []string{"D1"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},
	{
		ID: "d1.query", CLIPath: []string{"d1", "query"},
		WranglerEquivalent: "wrangler d1 query",
		Service:            "d1", Scope: "database", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database/{database_id}/query"},
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"unbounded_delete_guard"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.import", CLIPath: []string{"d1", "import"},
		WranglerEquivalent: "wrangler d1 import",
		Service:            "d1", Scope: "database", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database/{database_id}/import"},
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"dump_file_readable"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.export", CLIPath: []string{"d1", "export"},
		WranglerEquivalent: "wrangler d1 export",
		Service:            "d1", Scope: "database", Verb: "read",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database/{database_id}/export"},
		Permissions: Permissions{Account: []string{"D1"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.migrations.list", CLIPath: []string{"d1", "migrations", "list"},
		WranglerEquivalent: "wrangler d1 migrations list",
		Service:            "d1", Scope: "database", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/d1/database/{database_id}/query"},
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"migrations_dir_present"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.migrations.create", CLIPath: []string{"d1", "migrations", "create"},
		WranglerEquivalent: "wrangler d1 migrations create",
		Service:            "d1", Scope: "database", Verb: "write",
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"migrations_dir_writable"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.migrations.apply", CLIPath: []string{"d1", "migrations", "apply"},
		WranglerEquivalent: "wrangler d1 migrations apply",
		Service:            "d1", Scope: "database", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database/{database_id}/query"},
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"migrations_dir_present"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "d1.time-travel.restore", CLIPath: []string{"d1", "time-travel", "restore"},
		WranglerEquivalent: "wrangler d1 time-travel restore",
		Service:            "d1", Scope: "database", Verb: "write",
		APIOps: []string{
			"GET /accounts/{account_id}/d1/database/{database_id}/time_travel/bookmark",
			"POST /accounts/{account_id}/d1/database/{database_id}/restore",
		},
		Permissions: Permissions{Account: []string{"D1"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- d1 workflow commands (FEAT-018) ---
	{
		ID: "d1.parity", CLIPath: []string{"d1", "parity"},
		Service: "d1", Scope: "database", Verb: "read",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database/{database_id}/query"},
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"local_sqlite_readable"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// --- d1 workflow commands (FEAT-018) ---
	{
		ID: "d1.push-sql", CLIPath: []string{"d1", "push-sql"},
		Service: "d1", Scope: "database", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/d1/database/{database_id}/query"},
		Permissions: Permissions{Account: []string{"D1"}},
		LocalChecks: []string{"dump_file_readable"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- dns records (zone-scoped) ---
	{
		ID: "dns.create", CLIPath: []string{"dns", "create"},
		Service: "dns", Scope: "record", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/dns_records"},
		Permissions: Permissions{Zone: []string{"DNS"}},
		LocalChecks: []string{"record_type_known", "record_content_valid"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "dns.list", CLIPath: []string{"dns", "list"},
		Service: "dns", Scope: "zone", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/dns_records"},
		Permissions: Permissions{Zone: []string{"DNS"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "dns.get", CLIPath: []string{"dns", "get"},
		Service: "dns", Scope: "record", Verb: "read",
		APIOps:      []string{"GET /zones/{zone_id}/dns_records/{record_id}"},
		Permissions: Permissions{Zone: []string{"DNS"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "dns.update", CLIPath: []string{"dns", "update"},
		Service: "dns", Scope: "record", Verb: "write",
		APIOps:      []string{"PUT /zones/{zone_id}/dns_records/{record_id}"},
		Permissions: Permissions{Zone: []string{"DNS"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "dns.delete", CLIPath: []string{"dns", "delete"},
		Service: "dns", Scope: "record", Verb: "delete",
		APIOps:      []string{"DELETE /zones/{zone_id}/dns_records/{record_id}"},
		Permissions: Permissions{Zone: []string{"DNS"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- email (wave 3; permissions from the Qwen dataset) ---
	{
		ID: "email.rules.list", CLIPath: []string{"email", "rules", "list"},
		Service: "email", Scope: "zone", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/email/routing/rules"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "email.rules.get", CLIPath: []string{"email", "rules", "get"},
		Service: "email", Scope: "rule", Verb: "read",
		APIOps:      []string{"GET /zones/{zone_id}/email/routing/rules/{rule_id}"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "email.rules.create", CLIPath: []string{"email", "rules", "create"},
		Service: "email", Scope: "rule", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/email/routing/rules"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.rules.update", CLIPath: []string{"email", "rules", "update"},
		Service: "email", Scope: "rule", Verb: "write",
		APIOps:      []string{"PUT /zones/{zone_id}/email/routing/rules/{rule_id}"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.rules.delete", CLIPath: []string{"email", "rules", "delete"},
		Service: "email", Scope: "rule", Verb: "delete",
		APIOps:      []string{"DELETE /zones/{zone_id}/email/routing/rules/{rule_id}"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.destinations.list", CLIPath: []string{"email", "destinations", "list"},
		Service: "email", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/email/routing/addresses"},
		Permissions: Permissions{Account: []string{"Email Routing Addresses"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "email.destinations.add", CLIPath: []string{"email", "destinations", "add"},
		Service: "email", Scope: "address", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/email/routing/addresses"},
		Permissions: Permissions{Account: []string{"Email Routing Addresses"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.destinations.get", CLIPath: []string{"email", "destinations", "get"},
		Service: "email", Scope: "address", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/email/routing/addresses/{address_id}"},
		Permissions: Permissions{Account: []string{"Email Routing Addresses"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "email.destinations.delete", CLIPath: []string{"email", "destinations", "delete"},
		Service: "email", Scope: "address", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/email/routing/addresses/{address_id}"},
		Permissions: Permissions{Account: []string{"Email Routing Addresses"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.catchall.update", CLIPath: []string{"email", "catchall", "update"},
		Service: "email", Scope: "zone", Verb: "write",
		APIOps:      []string{"PUT /zones/{zone_id}/email/routing/rules/catch_all"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.settings", CLIPath: []string{"email", "settings"},
		Service: "email", Scope: "zone", Verb: "read",
		APIOps:      []string{"GET /zones/{zone_id}/email/routing"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "email.enable", CLIPath: []string{"email", "enable"},
		Service: "email", Scope: "zone", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/email/routing/enable"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "email.disable", CLIPath: []string{"email", "disable"},
		Service: "email", Scope: "zone", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/email/routing/disable"},
		Permissions: Permissions{Zone: []string{"Email Routing Rules"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- waf (wave 3; permissions from the Qwen dataset) ---
	{
		ID: "waf.packages", CLIPath: []string{"waf", "packages"},
		Service: "waf", Scope: "zone", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/firewall/packages"},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.rules", CLIPath: []string{"waf", "rules"},
		Service: "waf", Scope: "package", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/firewall/packages/{package_id}/rules"},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.ratelimit", CLIPath: []string{"waf", "ratelimit"},
		Service: "waf", Scope: "zone", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/rate_limits"},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		LocalChecks: []string{"path_expression_valid"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.rule", CLIPath: []string{"waf", "rule"},
		Service: "waf", Scope: "rule", Verb: "write",
		APIOps: []string{
			"GET /zones/{zone_id}/firewall/packages/{package_id}/rules/{rule_id}",
			"PATCH /zones/{zone_id}/firewall/packages/{package_id}/rules/{rule_id}",
		},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.access.list", CLIPath: []string{"waf", "access", "list"},
		Service: "waf", Scope: "zone", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/firewall/access_rules/rules"},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.access.create", CLIPath: []string{"waf", "access", "create"},
		Service: "waf", Scope: "rule", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/firewall/access_rules/rules"},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.access.delete", CLIPath: []string{"waf", "access", "delete"},
		Service: "waf", Scope: "rule", Verb: "delete",
		APIOps:      []string{"DELETE /zones/{zone_id}/firewall/access_rules/rules/{rule_id}"},
		Permissions: Permissions{Zone: []string{"Zone WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.list.ls", CLIPath: []string{"waf", "list", "ls"},
		Service: "waf", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/rules/lists"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.list.create", CLIPath: []string{"waf", "list", "create"},
		Service: "waf", Scope: "list", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/rules/lists"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.list.update", CLIPath: []string{"waf", "list", "update"},
		Service: "waf", Scope: "list", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/rules/lists/{list_id}"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.list.delete", CLIPath: []string{"waf", "list", "delete"},
		Service: "waf", Scope: "list", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/rules/lists/{list_id}"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},
	{
		ID: "waf.list.item.ls", CLIPath: []string{"waf", "list", "item", "ls"},
		Service: "waf", Scope: "list", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/rules/lists/{list_id}/items"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.list.item.add", CLIPath: []string{"waf", "list", "item", "add"},
		Service: "waf", Scope: "list", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/rules/lists/{list_id}/items"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.list.item.replace", CLIPath: []string{"waf", "list", "item", "replace"},
		Service: "waf", Scope: "list", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/rules/lists/{list_id}/items"},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "waf.managed-ruleset.update", CLIPath: []string{"waf", "managed-ruleset", "update"},
		Service: "waf", Scope: "phase", Verb: "write",
		APIOps: []string{
			"GET /accounts/{account_id}/rulesets/phases/{phase}/entrypoint",
			"PUT /accounts/{account_id}/rulesets/phases/{phase}/entrypoint",
		},
		Permissions: Permissions{Account: []string{"Account WAF"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- ssl (wave 3; permissions from the Qwen dataset) ---
	{
		ID: "ssl.status", CLIPath: []string{"ssl", "status"},
		Service: "ssl", Scope: "zone", Verb: "read",
		APIOps:      []string{"GET /zones/{zone_id}/settings/ssl"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.settings", CLIPath: []string{"ssl", "settings"},
		Service: "ssl", Scope: "zone", Verb: "read",
		APIOps: []string{
			"GET /zones/{zone_id}/settings",
			"GET /zones/{zone_id}/ssl/universal/settings",
		},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.update", CLIPath: []string{"ssl", "update"},
		Service: "ssl", Scope: "zone", Verb: "write",
		APIOps: []string{
			"PATCH /zones/{zone_id}/settings/ssl",
			"PATCH /zones/{zone_id}/settings",
		},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.verify", CLIPath: []string{"ssl", "verify"},
		Service: "ssl", Scope: "zone", Verb: "read",
		APIOps:      []string{"GET /zones/{zone_id}/ssl/verification"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.custom-hostname.list", CLIPath: []string{"ssl", "custom-hostname", "list"},
		Service: "ssl", Scope: "zone", Verb: "list",
		APIOps:      []string{"GET /zones/{zone_id}/custom_hostnames"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.custom-hostname.get", CLIPath: []string{"ssl", "custom-hostname", "get"},
		Service: "ssl", Scope: "hostname", Verb: "read",
		APIOps:      []string{"GET /zones/{zone_id}/custom_hostnames/{hostname_id}"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.custom-hostname.create", CLIPath: []string{"ssl", "custom-hostname", "create"},
		Service: "ssl", Scope: "hostname", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/custom_hostnames"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.custom-hostname.update", CLIPath: []string{"ssl", "custom-hostname", "update"},
		Service: "ssl", Scope: "hostname", Verb: "write",
		APIOps:      []string{"PATCH /zones/{zone_id}/custom_hostnames/{hostname_id}"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "ssl.custom-hostname.delete", CLIPath: []string{"ssl", "custom-hostname", "delete"},
		Service: "ssl", Scope: "hostname", Verb: "delete",
		APIOps:      []string{"DELETE /zones/{zone_id}/custom_hostnames/{hostname_id}"},
		Permissions: Permissions{Zone: []string{"SSL and Certificates"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- cache (wave 3; permissions from the Qwen dataset) ---
	{
		ID: "cache.purge", CLIPath: []string{"cache", "purge"},
		Service: "cache", Scope: "zone", Verb: "write",
		APIOps:      []string{"POST /zones/{zone_id}/purge_cache"},
		Permissions: Permissions{Zone: []string{"Cache Purge"}},
		LocalChecks: []string{"force_flag_for_purge_all"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "cache.settings", CLIPath: []string{"cache", "settings"},
		Service: "cache", Scope: "zone", Verb: "write",
		APIOps: []string{
			"GET /zones/{zone_id}/settings",
			"PATCH /zones/{zone_id}/settings",
		},
		Permissions: Permissions{Zone: []string{"Cache Purge"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},

	// --- hyperdrive (wave 3; permissions from the Qwen dataset) ---
	{
		ID: "hyperdrive.list", CLIPath: []string{"hyperdrive", "list"},
		WranglerEquivalent: "wrangler hyperdrive list",
		Service:            "hyperdrive", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/hyperdrive/configs"},
		Permissions: Permissions{Account: []string{"Hyperdrive"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "hyperdrive.get", CLIPath: []string{"hyperdrive", "get"},
		Service: "hyperdrive", Scope: "config", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/hyperdrive/configs/{config_id}"},
		Permissions: Permissions{Account: []string{"Hyperdrive"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "hyperdrive.create", CLIPath: []string{"hyperdrive", "create"},
		WranglerEquivalent: "wrangler hyperdrive create",
		Service:            "hyperdrive", Scope: "config", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/hyperdrive/configs"},
		Permissions: Permissions{Account: []string{"Hyperdrive"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "hyperdrive.update", CLIPath: []string{"hyperdrive", "update"},
		WranglerEquivalent: "wrangler hyperdrive update",
		Service:            "hyperdrive", Scope: "config", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/hyperdrive/configs/{config_id}"},
		Permissions: Permissions{Account: []string{"Hyperdrive"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "hyperdrive.delete", CLIPath: []string{"hyperdrive", "delete"},
		WranglerEquivalent: "wrangler hyperdrive delete",
		Service:            "hyperdrive", Scope: "config", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/hyperdrive/configs/{config_id}"},
		Permissions: Permissions{Account: []string{"Hyperdrive"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},

	// --- alerts (wave 3; permissions from the Qwen dataset) ---
	// AlertService is file-backed (local rules + history) — it makes no
	// Cloudflare API calls, so no permission applies.
	{
		ID: "alerts.list", CLIPath: []string{"alerts", "list"},
		Service: "alerts", Scope: "config", Verb: "list",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.get", CLIPath: []string{"alerts", "get"},
		Service: "alerts", Scope: "rule", Verb: "read",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.create", CLIPath: []string{"alerts", "create"},
		Service: "alerts", Scope: "rule", Verb: "write",
		LocalChecks: []string{"condition_known", "threshold_numeric"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.update", CLIPath: []string{"alerts", "update"},
		Service: "alerts", Scope: "rule", Verb: "write",
		LocalChecks: []string{"condition_known", "threshold_numeric"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.delete", CLIPath: []string{"alerts", "delete"},
		Service: "alerts", Scope: "rule", Verb: "delete",
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.test", CLIPath: []string{"alerts", "test"},
		Service: "alerts", Scope: "rule", Verb: "read",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.history", CLIPath: []string{"alerts", "history"},
		Service: "alerts", Scope: "config", Verb: "list",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "alerts.check", CLIPath: []string{"alerts", "check"},
		Service: "alerts", Scope: "config", Verb: "read",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// --- tunnel (wave 4; permissions from the Qwen dataset) ---
	// cloudflared/Wrangler have no stable REST-tunnel equivalent commands, so
	// WranglerEquivalent is left empty for the whole group.
	{
		ID: "tunnel.list", CLIPath: []string{"tunnel", "list"},
		Service: "tunnel", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/cfd_tunnel"},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "tunnel.get", CLIPath: []string{"tunnel", "get"},
		Service: "tunnel", Scope: "tunnel", Verb: "read",
		APIOps: []string{
			// name lookup goes through the list call before the direct get
			"GET /accounts/{account_id}/cfd_tunnel",
			"GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}",
		},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "tunnel.create", CLIPath: []string{"tunnel", "create"},
		Service: "tunnel", Scope: "tunnel", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/cfd_tunnel"},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "tunnel.delete", CLIPath: []string{"tunnel", "delete"},
		Service: "tunnel", Scope: "tunnel", Verb: "delete",
		APIOps: []string{
			// cascade=true tears down live connections before the delete
			"DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}/connections",
			"DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}",
		},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},
	{
		ID: "tunnel.token", CLIPath: []string{"tunnel", "token"},
		Service: "tunnel", Scope: "tunnel", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/token"},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "tunnel.connections", CLIPath: []string{"tunnel", "connections"},
		Service: "tunnel", Scope: "tunnel", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/connections"},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "tunnel.cleanup", CLIPath: []string{"tunnel", "cleanup"},
		Service: "tunnel", Scope: "tunnel", Verb: "delete",
		APIOps: []string{
			"DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}/connections",
			"DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}",
		},
		Permissions: Permissions{Account: []string{"Cloudflare Tunnel"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},

	// --- account (wave 4) ---
	// The profile subcommands (list/add/switch/remove/current) are local
	// config-file operations — no Cloudflare API calls, no permission applies.
	{
		ID: "account.list", CLIPath: []string{"account", "list"},
		Service: "account", Scope: "config", Verb: "list",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "account.add", CLIPath: []string{"account", "add"},
		Service: "account", Scope: "config", Verb: "write",
		LocalChecks: []string{"account_id_present", "api_token_present"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "account.switch", CLIPath: []string{"account", "switch"},
		Service: "account", Scope: "config", Verb: "write",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "account.remove", CLIPath: []string{"account", "remove"},
		// removes the local profile only; no Cloudflare resource is touched
		Service: "account", Scope: "config", Verb: "delete",
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "account.current", CLIPath: []string{"account", "current"},
		Service: "account", Scope: "config", Verb: "read",
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		// any token can verify itself; no account-level permission is checked
		ID: "account.verify", CLIPath: []string{"account", "verify"},
		Service: "account", Scope: "config", Verb: "read",
		APIOps:      []string{"GET /user/tokens/verify"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// member/role commands; "Memberships" is a user-scoped permission in the
	// dataset (docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md).
	{
		ID: "account.member.list", CLIPath: []string{"account", "member", "list"},
		Service: "account", Scope: "member", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/members"},
		Permissions: Permissions{User: []string{"Memberships"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "account.member.invite", CLIPath: []string{"account", "member", "invite"},
		Service: "account", Scope: "member", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/members"},
		Permissions: Permissions{User: []string{"Memberships"}},
		LocalChecks: []string{"email_format", "role_ids_present"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "account.member.update", CLIPath: []string{"account", "member", "update"},
		Service: "account", Scope: "member", Verb: "write",
		APIOps:      []string{"PUT /accounts/{account_id}/members/{member_id}"},
		Permissions: Permissions{User: []string{"Memberships"}},
		LocalChecks: []string{"role_ids_present"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "account.member.remove", CLIPath: []string{"account", "member", "remove"},
		Service: "account", Scope: "member", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/members/{member_id}"},
		Permissions: Permissions{User: []string{"Memberships"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},
	{
		ID: "account.role.list", CLIPath: []string{"account", "role", "list"},
		Service: "account", Scope: "role", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/roles"},
		Permissions: Permissions{User: []string{"Memberships"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// Cloudflare account audit log; per the dataset's wave-3 mapping the
	// /accounts/{id}/logs/audit surface requires "Account Settings" (Read or
	// Edit) — "Access: Audit Logs" in the dataset is the Zero Trust surface.
	{
		ID: "account.audit-logs.list", CLIPath: []string{"account", "audit-logs", "list"},
		Service: "account", Scope: "audit_log", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/logs/audit"},
		Permissions: Permissions{Account: []string{"Account Settings"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "account.audit-logs.history", CLIPath: []string{"account", "audit-logs", "history"},
		Service: "account", Scope: "audit_log", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/logs/audit/{log_id}/history"},
		Permissions: Permissions{Account: []string{"Account Settings"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},

	// --- logpush (FEAT-037 wave 1; permissions from the Qwen dataset) ---
	// Wrangler/cloudflared have no Logpush equivalent commands, so
	// WranglerEquivalent is left empty for the whole group — this surface is
	// the CLI's differentiation. The dataset maps every Logpush operation
	// (reads included) to account "Logs: Edit".
	{
		ID: "logpush.job.create", CLIPath: []string{"logpush", "job", "create"},
		Service: "logpush", Scope: "job", Verb: "write",
		APIOps:      []string{"POST /accounts/{account_id}/logpush/jobs"},
		Permissions: Permissions{Account: []string{"Logs"}},
		LocalChecks: []string{"dataset_present", "destination_conf_present"},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "logpush.job.list", CLIPath: []string{"logpush", "job", "list"},
		Service: "logpush", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/logpush/jobs"},
		Permissions: Permissions{Account: []string{"Logs"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "logpush.job.get", CLIPath: []string{"logpush", "job", "get"},
		Service: "logpush", Scope: "job", Verb: "read",
		APIOps:      []string{"GET /accounts/{account_id}/logpush/jobs/{job_id}"},
		Permissions: Permissions{Account: []string{"Logs"}},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "logpush.job.update", CLIPath: []string{"logpush", "job", "update"},
		Service: "logpush", Scope: "job", Verb: "write",
		APIOps: []string{
			// update is a full replace in the API: the current job is
			// fetched first and the partial changes merged on top
			"GET /accounts/{account_id}/logpush/jobs/{job_id}",
			"PUT /accounts/{account_id}/logpush/jobs/{job_id}",
		},
		Permissions: Permissions{Account: []string{"Logs"}},
		DangerLevel: "medium", Destructive: false, Trackable: true,
	},
	{
		ID: "logpush.job.delete", CLIPath: []string{"logpush", "job", "delete"},
		Service: "logpush", Scope: "job", Verb: "delete",
		APIOps:      []string{"DELETE /accounts/{account_id}/logpush/jobs/{job_id}"},
		Permissions: Permissions{Account: []string{"Logs"}},
		DangerLevel: "high", Destructive: true, Trackable: true,
	},
	{
		ID: "logpush.ownership.verify", CLIPath: []string{"logpush", "ownership", "verify"},
		Service: "logpush", Scope: "destination", Verb: "read",
		APIOps: []string{
			"POST /accounts/{account_id}/logpush/ownership",
			"POST /accounts/{account_id}/logpush/ownership/validate",
		},
		Permissions: Permissions{Account: []string{"Logs"}},
		LocalChecks: []string{"destination_conf_present"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
}
