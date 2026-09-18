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
		Service: "r2", Scope: "account", Verb: "write",
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
		Service: "r2", Scope: "account", Verb: "list",
		APIOps:      []string{"GET /accounts/{account_id}/r2/buckets"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.bucket.get", CLIPath: []string{"r2", "bucket", "get"},
		WranglerEquivalent: "wrangler r2 bucket info",
		Service: "r2", Scope: "bucket", Verb: "read",
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
		Service: "r2", Scope: "bucket", Verb: "delete",
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
		Service: "r2", Scope: "object", Verb: "write",
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
		Service: "r2", Scope: "object", Verb: "read",
		APIOps:      []string{"GET /{bucket}/{key}"},
		Permissions: Permissions{Account: []string{"Workers R2 Storage"}},
		RateLimit:   &RateLimitBehavior{HTTPStatus: 429, RetryAfter: true, Bucket: "r2_rest_or_s3"},
		DangerLevel: "low", Destructive: false, Trackable: true,
	},
	{
		ID: "r2.object.delete", CLIPath: []string{"r2", "object", "delete"},
		WranglerEquivalent: "wrangler r2 object delete",
		Service: "r2", Scope: "object", Verb: "delete",
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
}
