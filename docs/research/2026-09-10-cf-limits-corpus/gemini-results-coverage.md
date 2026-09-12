# Gemini deep-research results — platform coverage completion (pillar A)

Provenance: deep research run 2026-09-12 from prompt 2 in
deep-research-prompts.md (coverage completion: DO, Workflows, Tunnels,
Gateway/WARP, long-tail zone services, account/org/audit, billing API).
Captured via clipboard — NOTE: the paste flattened all markdown structure
(no headings/fences survived); JSON block recovered programmatically.
Feeds the next depth-pack features (DO/Workflows/Tunnels command surface,
Zero Trust roadmap item, audit-log feature, FEAT-034+ candidates).

---

Programmatic Control Plane Architecture for Enterprise Cloudflare InfrastructureFull-account programmatic control across Cloudflare requires navigating a structural dichotomy between edge-isolate data planes and centralized control planes. Enterprise infrastructure automation bifurcates into two distinct operational patterns: compute services (Workers, Workflows) provide deep dual-plane parity across REST APIs and the Wrangler CLI, whereas perimeter security, ingress routing, zero-trust connectivity, and corporate governance operate almost exclusively through the global Cloudflare v4 REST API. Building resilient orchestration pipelines requires accounting for this functional asymmetry, mapping exact API token permission boundaries, and architecting around Cloudflare's baseline API rate limits.The baseline rate limit across the Cloudflare Client API is 1,200 requests per 5-minute rolling window per user or account token, coupled with an ingress constraint of 200 requests per second per edge IP address. Token-level limits apply cumulatively across all mutations and queries, regardless of whether requests originate from the Cloudflare Dashboard, CI/CD runners, Terraform execution nodes, or custom orchestration daemons. Certain sub-resource controllers enforce independent rate budgets: Gateway Lists, Ruleset updates, Cache Purges, and GraphQL Analytics queries (which throttle dynamically based on query complexity score up to 320 requests per 5 minutes). While Enterprise organizations can negotiate custom rate-quota elevations through Cloudflare Support, production automation frameworks must enforce token-bucket throttling, backoff-retry handlers, and asynchronous event streaming to prevent HTTP 429 rate-limiting lockouts.Executing programmatic account control requires understanding the boundary between automatable control surfaces and manual dashboard operations across the seven core platform domains: stateful compute isolates, workflow engines, edge ingress tunnels, Zero Trust device posture and proxy engines, edge performance and application security suites, granular IAM governance with audit log pipelines, and financial clearinghouse interfaces.Durable Objects Namespace Management and Scripting BoundariesDurable Objects enforce an architectural boundary between the in-worker runtime scripting API and the administrative REST control plane. The runtime scripting API executes strictly within the V8 isolate environment on Cloudflare's distributed edge. Inside a Worker, an application accesses a stateful object by generating or deriving a 64-character hexadecimal unique identifier using idFromName(name), idFromString(id), or newUniqueId(), then obtaining a stub proxy via env.NAMESPACE.get(id). Once instantiated, all data-plane operations—including transactional reads and writes, SQLite storage operations (ctx.storage.sql.exec), transactional key-value interactions (ctx.storage.get, ctx.storage.put), persistent alarms (ctx.storage.setAlarm), and WebSocket hibernation listeners—execute exclusively within the isolate runtime. The internal state of a Durable Object lives on local NVMe disk and in memory on the server where the object is actively coordinated.The external REST management API cannot read, write, or query the internal storage or active memory of a live Durable Object. Cloudflare exposes no REST endpoints for executing ad-hoc SQL statements against an object's SQLite database or fetching key-value pairs stored within an instance. To programmatically inspect, export, or modify internal object state from an external orchestrator, engineers must deploy a custom Worker proxy script. This Worker binds directly to the Durable Object namespace, handles inbound authenticated HTTP requests, routes requests to the appropriate instance stub via RPC or fetch(), and serializes the internal storage state back over the HTTP response.The external REST API serves solely as a control-plane inventory surface. It allows automation pipelines to enumerate account namespaces, create new namespace bindings, and discover live, actively allocated object instances using cursor-based pagination. Registering new Durable Object classes, enabling the SQLite storage engine backend, and applying database schema migrations are executed through Worker deployment bundles. Migrations must be defined in the Worker configuration file (wrangler.toml or wrangler.jsonc) using migration directives such as new_sqlite_classes, renamed_classes, deleted_classes, and transferred_classes, and deployed via multipart Worker script deployments.Wrangler provides zero CLI commands for inspecting active instances, querying storage keys, or modifying live object state. Wrangler's coverage is confined entirely to script packaging, migration tracking, and binding declarations during wrangler deploy. All Durable Object REST management endpoints operate at the account scope and require API tokens configured with Workers Scripts: Read or Workers Scripts: Edit permissions. Requests adhere to the platform's global rate limit of 1,200 requests per 5 minutes.OperationMethod + EndpointScopeWrangler?NotesList NamespacesGET /accounts/{account_id}/workers/durable_objects/namespacesAccount (Workers Scripts: Read)NoLists all registered Durable Object namespaces, class associations, and bound Worker script names.Create NamespacePOST /accounts/{account_id}/workers/durable_objects/namespacesAccount (Workers Scripts: Edit)NoManually provisions a namespace binding identifier outside of automatic script upload bundling.List Live Object InstancesGET /accounts/{account_id}/workers/durable_objects/namespaces/{namespace_id}/objectsAccount (Workers Scripts: Read)NoCursor-paginated enumeration of currently allocated, non-evicted Durable Object instance IDs.Apply Schema MigrationsPUT /accounts/{account_id}/workers/scripts/{script_name}Account (Workers Scripts: Edit)Yes (wrangler deploy)Executes database schema migrations (new_sqlite_classes, transferred_classes) during Worker deployment.Query Operational TelemetryPOST /client/v4/graphql (Dataset: durableObjectsInvocationsAdaptiveGroups)Account (Analytics: Read)NoAggregates operational metrics: CPU wall time, storage read/write units, memory duration, and request rates.Durable Execution and State Lifecycle in Cloudflare WorkflowsCloudflare Workflows provides durable execution for multi-step, distributed applications running on Workers infrastructure. In contrast to the read-only operational boundaries of Durable Objects, Workflows provides symmetric feature parity across its REST control plane, the Wrangler CLI, and its in-worker TypeScript SDK (cloudflare:workers). Workflows guarantees at-least-once step execution, handles automated state checkpoints between non-blocking steps (step.do), serializes step return data, and supports deterministic, sleep-driven scheduling (step.sleep) without keeping the underlying compute isolate continuously active.The REST API exposes complete lifecycle management over workflow definitions and individual runtime instances. At the definition layer, automation pipelines can enumerate all deployed workflows, inspect scheduled cron triggers, and retrieve aggregated instance execution tallies (queued, running, paused, errored, terminated, complete). At the instance layer, the REST API enables full programmatic dispatch and operational remediation. Beyond initiating single runs with custom payloads (POST /instances), platforms can perform bulk provisioning via /instances/batch to enqueue batches of instances with configurable data retention windows.Operational remediation of executing instances is handled by issuing a PATCH request to /instances/{instance_id}/status. This endpoint allows automation controllers to pause long-running instances, resume paused workflows, terminate misbehaving or looping tasks, and restart failed executions from the last successful checkpoint. Granular execution logs, step durations, and serialized intermediate step outputs can be queried via /instances/{instance_id} and /instances/{instance_id}/step.The Wrangler CLI provides first-class support for Workflows through the wrangler workflows command family (supported in Wrangler v3.83.0 and above). Developers can execute wrangler workflows list, trigger instances with JSON payloads (wrangler workflows trigger <name> --params '<json>'), query run state (wrangler workflows instances describe <name> <id>), modify instance execution state (pause, resume, terminate), and dispatch asynchronous events into running instances using wrangler workflows instances send-event.Wrangler also includes a local execution environment. Appending the --local flag redirects workflow operations to a local Miniflare/workerd dev session (defaulting to port 8787), providing local step execution, database persistence, and event triggers without provisioning cloud resources. All Workflows REST endpoints are account-scoped, consume permissions from Workers Scripts: Read or Workers Scripts: Edit, and share the baseline rate limit of 1,200 requests per 5 minutes.OperationMethod + EndpointScopeWrangler?NotesList Workflow DefinitionsGET /accounts/{account_id}/workflowsAccount (Workers Scripts: Read)Yes (wrangler workflows list)Returns all deployed workflows, associated entry classes, cron schedules, and instance health aggregates.Trigger Workflow InstancePOST /accounts/{account_id}/workflows/{workflow_name}/instancesAccount (Workers Scripts: Edit)Yes (wrangler workflows trigger)Dispatches a single execution instance, accepting an optional custom instance_id and a JSON parameters payload.Bulk Provision InstancesPOST /accounts/{account_id}/workflows/{workflow_name}/instances/batchAccount (Workers Scripts: Edit)NoConcurrently creates multiple workflow instances with custom payloads and step data retention rules.Fetch Instance Execution LogsGET /accounts/{account_id}/workflows/{workflow_name}/instances/{instance_id}Account (Workers Scripts: Read)Yes (wrangler workflows instances describe)Retrieves granular execution metadata, including run status, timestamps, errors, retries, and step details.Modify Instance Lifecycle StatusPATCH /accounts/{account_id}/workflows/{workflow_name}/instances/{instance_id}/statusAccount (Workers Scripts: Edit)Yes (wrangler workflows instances [pause|resume|terminate])Dynamically pauses, resumes, restarts, or terminates an active or queued workflow instance.Zero Trust Tunnel Architecture and Private Routing TopologyCloudflare Tunnels establishes outbound, encrypted connections between private enterprise networks and the Cloudflare edge without requiring public IP allocations or open inbound firewall ports. The tunnel ecosystem has shifted from locally configured daemons to remotely managed tunnels managed via the Cloudflare control plane. In a legacy local deployment, the cloudflared daemon reads a local YAML configuration file defining ingress rules, credentials files, and TLS settings. In remotely managed deployments, cloudflared is deployed as a stateless connector using an account-level provisioning token (cloudflared tunnel run --token <TOKEN>); the daemon dynamically retrieves its ingress configurations, private network routes, and transport settings directly from the Cloudflare API.The REST management API provides complete CRUD control over tunnel identities, ingress rule routing trees, private network routes, and Virtual Networks (VNets). Exposing a public application through a remotely managed tunnel requires two distinct operations: first, pushing the ingress route array via PUT /cfd_tunnel/{tunnel_id}/configurations to map the public hostname to an internal service URL (e.g., http://10.0.4.15:8080), and second, creating a DNS CNAME record on the target zone pointing the public hostname to the tunnel's edge target (<tunnel_id>.cfargotunnel.com).Connecting private corporate networks is handled through the Zero Trust Teamnet routing engine. Cloudflare has deprecated the legacy URL-encoded path format (/teamnet/routes/network/{ip_network_encoded}) in favor of standard REST endpoints (/teamnet/routes and /teamnet/routes/{route_id}), passing the CIDR block within the JSON request body. To prevent routing conflicts across overlapping RFC1918 CIDR blocks, organizations can provision isolated Virtual Networks via /teamnet/virtual_networks and bind CIDR routes to specific VNets.For site-to-site connectivity and bidirectional routing across remote server clusters, Cloudflare exposes the WARP Connector endpoint (/warp_connector). WARP Connector registers dedicated tunnel endpoints that route traffic directly between client fleets running the WARP agent and internal subnet nodes without requiring client-side application proxies.Wrangler contains no configuration or deployment coverage for Cloudflare Tunnels. All CLI management must be executed through the dedicated cloudflared binary (cloudflared tunnel create, cloudflared tunnel route ip add, cloudflared tunnel token) or programmatically via the REST API or Terraform/OpenTofu. Managing tunnels via the API requires account-level permissions (Cloudflare Tunnel: Read, Cloudflare Tunnel: Edit), while provisioning public hostnames requires zone-level DNS permissions (DNS: Edit) on the target zone. Granular IAM roles can also restrict a principal's administrative scope to specific tunnels rather than granting broad, account-wide Zero Trust permissions. Standard rate limits of 1,200 requests per 5 minutes apply across all tunnel endpoints.OperationMethod + EndpointScopeWrangler?NotesProvision Remote TunnelPOST /accounts/{account_id}/cfd_tunnelAccount (Cloudflare Tunnel: Edit)No (cloudflared tunnel create)Creates a new tunnel identity with a generated UUID and secret key; returns the tunnel identifier.Fetch Tunnel Run TokenGET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/tokenAccount (Cloudflare Tunnel: Read)No (cloudflared tunnel token)Retrieves the base64-encoded token passed to cloudflared daemons for remote configuration management.Update Remote Ingress ConfigPUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurationsAccount (Cloudflare Tunnel: Edit)NoPushes the ingress rules tree, defining public hostnames, origin transport protocols, and fallback routes.Provision Virtual NetworkPOST /accounts/{account_id}/teamnet/virtual_networksAccount (Cloudflare Tunnel: Edit)No (cloudflared tunnel vnet add)Provisions an isolated virtual routing domain to segment overlapping private CIDR spaces.Create Private CIDR RoutePOST /accounts/{account_id}/teamnet/routesAccount (Cloudflare Tunnel: Edit)No (cloudflared tunnel route ip add)Binds a private IP network CIDR to a specific tunnel and virtual network ID using the body-driven schema.Zero Trust Gateway Policy Enforcement and Fleet Posture GovernanceCloudflare Zero Trust Gateway acts as an edge security proxy, evaluating and enforcing security policies across outbound corporate traffic via DNS filtering, HTTP inspection, Network (TCP/UDP) firewalls, and dedicated Egress IP steering. Fleet devices connect to Gateway using the Cloudflare WARP client application, which establishes an encrypted WireGuard-based transport tunnel to the nearest edge data center.Policy evaluation is managed through the Gateway Rules API (/accounts/{account_id}/gateway/rules). Rules define granular conditions using Cloudflare's wirefilter expression syntax, evaluating parameters such as user identity (SAML/IdP claims resolved via Cloudflare Access), source IP, destination SNI, URL categories, or file download signatures. Matching actions include block, allow, isolate (routing traffic through remote Browser Isolation sessions), and l4_override (steering traffic through dedicated egress IP proxies).Context-aware conditional access is governed by device posture rules (/accounts/{account_id}/devices/posture). Posture rules assess client-side security signals before granting access to network resources or SaaS applications. Posture checks evaluate native OS telemetry (disk encryption status, firewall configuration, OS build version, domain membership, running processes, or client certificate validation).To incorporate external device telemetry, the posture engine integrates with enterprise Endpoint Detection and Response (EDR) and Unified Endpoint Management (UEM) platforms via /accounts/{account_id}/devices/posture/integration. Organizations can link platforms such as CrowdStrike Falcon, SentinelOne, Microsoft Intune, and VMware Workspace ONE, allowing Gateway to evaluate real-time zero-trust posture using dynamic machine risk scores.WARP client behavior across the fleet is controlled through device settings profiles (/accounts/{account_id}/devices/policy). Profiles manage administrative password overrides, captive portal detection timeouts, local domain fallbacks, and Split Tunnel include/exclude route tables. Split Tunneling routes (/accounts/{account_id}/devices/policy/{policy_id}/exclude) instruct the client whether specific network traffic should pass through the encrypted WARP tunnel or break out directly through the user's local network gateway. When a device is decommissioned or compromised, administrators can revoke its registration via /accounts/{account_id}/devices/{device_id}, which invalidates its client certificates and severs its connection to Gateway.Wrangler contains no configuration capabilities for Zero Trust Gateway or WARP. While end-user machines run a local warp-cli client for local debugging and registration status, fleet management is performed entirely via the REST API, the Zero Trust dashboard, or the Cloudflare Terraform provider. All Gateway and WARP endpoints operate at the account scope, requiring Zero Trust: Read or Zero Trust: Edit API token permissions. Standard rate limits of 1,200 requests per 5 minutes apply, while the Gateway Lists sub-API enforces specialized rate quotas for bulk indicator updates.OperationMethod + EndpointScopeWrangler?NotesUpsert Gateway RulePOST /accounts/{account_id}/gateway/rulesAccount (Zero Trust: Edit)NoProvisions dynamic filtering rules across DNS, HTTP, Network, or Egress inspection engines.Configure Split Tunnel ExclusionsPUT /accounts/{account_id}/devices/policy/{policy_id}/excludeAccount (Zero Trust: Edit)NoUpdates the Split Tunnel route table, specifying CIDRs, hostnames, or applications that bypass WARP encapsulation.Provision Device Posture RulePOST /accounts/{account_id}/devices/postureAccount (Zero Trust: Edit)NoConfigures endpoint compliance criteria (disk encryption, firewall status, OS version, file/registry existence).Register EDR Posture IntegrationPOST /accounts/{account_id}/devices/posture/integrationAccount (Zero Trust: Edit)NoConnects third-party telemetry providers (CrowdStrike, Intune, SentinelOne) for continuous endpoint risk evaluation.Revoke Device RegistrationDELETE /accounts/{account_id}/devices/{device_id}Account (Zero Trust: Edit)NoRevokes an enrolled WARP client instance, invalidating its session certificates and disconnecting it from Gateway.Edge Traffic Steering, Threat Mitigation, and Observability PipelinesManaging enterprise web applications at scale requires configuring a suite of edge performance, application security, and logging services. These capabilities span both account-wide shared infrastructure and zone-specific edge configurations, as detailed across the eight services below.Waiting RoomDeployed at the zone level, Waiting Room sits upstream of origin web servers to protect backend infrastructure from traffic spikes by queuing excess visitors in an edge-managed virtual waiting room. The /zones/{zone_id}/waiting_rooms API manages total active user thresholds, new visitor admission rates, session durations, and cryptographic cookie validation. The /events sub-resource configures scheduled surge events for planned launches, while /rules enables custom queuing expressions to bypass or force-queue specific HTTP requests based on URL paths, IP subnets, or request headers.SpectrumSpectrum provides reverse-proxy TCP and UDP protection, shielding non-HTTP protocols (SSH, SFTP, gaming backends, custom binary services) against volumetric DDoS attacks while providing edge TLS termination. Managed at the zone level via /zones/{zone_id}/spectrum/apps, Spectrum configurations map external port ranges and protocol listeners to origin server pools, configure Proxy Protocol headers to preserve client source IPs, and apply IP access rules directly to raw network streams.Load BalancingCloudflare Load Balancing provides multi-origin traffic distribution, geographic routing, and automated origin health failover. Control is split between the account and zone scopes: reusable Origin Pools (/accounts/{account_id}/load_balancers/pools) and Health Monitors (/accounts/{account_id}/load_balancers/monitors) are defined at the account level, while zone Load Balancers (/zones/{zone_id}/load_balancers) reference those pool IDs to manage ingress traffic, steering policies (e.g., Geo, Dynamic Latency, Proximity), and failover thresholds.Page ShieldPage Shield protects client-side web applications against supply chain attacks, Magecart exploits, and unauthorized script tampering. The zone endpoint /zones/{zone_id}/page_shield enables passive detection, while /scripts and /connections catalog external script dependencies, monitor resource integrity, and assign machine-learning risk scores. The /policies sub-resource generates and enforces Content Security Policies (CSPs) to block unauthorized scripts and illicit outbound data exfiltration.Bot ManagementEnterprise Bot Management mitigates automated scraping, credential stuffing, and Layer 7 abuse without disrupting legitimate user traffic. Managed at the zone level via /zones/{zone_id}/bot_management, the API configures behavioral analysis engines, automated machine-learning scoring models, Super Bot Fight Mode (SBFM), JavaScript detections, and managed robots.txt enforcement. Ingress requests receive a bot score from 1 to 99, which custom WAF rules can evaluate to challenge or drop suspected bot traffic.TurnstileTurnstile provides an API-driven, privacy-preserving alternative to traditional CAPTCHAs. Configured at the account level via /accounts/{account_id}/challenges/widgets, Turnstile provisions frontend embed keys (sitekey) and backend validation secrets, supports multiple challenge modes (Managed, Non-Interactive, Invisible), and exposes secret-key rotation endpoints to ensure automated credential hygiene.Web AnalyticsCloudflare Web Analytics (RUM) collects privacy-focused browser performance metrics without using client-side cookies or tracking user identities. It is managed at the account level via /accounts/{account_id}/rum/site_info. Beyond provisioning tracking tokens, operators use the /rules endpoint to exclude corporate IP ranges, strip sensitive query parameters from logged paths, and customize Core Web Vitals telemetry aggregations.Logpush JobsLogpush delivers high-throughput, real-time log pipelines from Cloudflare edge nodes directly to external storage and analytics destinations, including Amazon S3, Google Cloud Storage, Microsoft Azure Blob, Apache Kafka, Datadog, Splunk, and Cloudflare R2. Logpush can be configured at either the zone level (for HTTP requests, firewall events) or the account level (for Zero Trust Gateway logs, Access authentication traces, and Audit Logs). The /logpush/jobs API handles ownership validation challenges, delivery filtering expressions, and sampled field selections.None of these eight services are supported in Wrangler; automation relies entirely on REST APIs, native SDKs, and Terraform providers. API token permissions require appropriate account-level (Load Balancing: Edit, Turnstile: Edit, Logs: Edit, Analytics: Edit) or zone-level (Waiting Room: Edit, Spectrum: Edit, Bot Management: Edit, Page Shield: Edit, Load Balancing: Edit, Logs: Edit) scopes. All endpoints adhere to the standard rate limit of 1,200 requests per 5 minutes.OperationMethod + EndpointScopeWrangler?NotesUpsert Load Balancer Origin PoolPOST /accounts/{account_id}/load_balancers/poolsAccount (Load Balancing: Edit)NoProvisions an origin pool defining backend IP addresses, health monitors, and notification thresholds.Update Bot Management ConfigPUT /zones/{zone_id}/bot_managementZone (Bot Management: Edit)NoConfigures Super Bot Fight Mode, machine-learning score generation, static resource protection, and JS detections.Enforce Page Shield CSP PolicyPOST /zones/{zone_id}/page_shield/policiesZone (Page Shield: Edit)NoDeploys Content Security Policies to block unauthorized scripts and destination connections.Provision Turnstile WidgetPOST /accounts/{account_id}/challenges/widgetsAccount (Turnstile: Edit)NoProvisions a Turnstile challenge widget, generating the public sitekey and backend verification secret.Provision Logpush Job PipelinePOST /accounts/{account_id}/logpush/jobsAccount or Zone (Logs: Edit)NoConfigures an automated log stream targeting object storage or SIEM endpoints, defining datasets, filters, and fields.Identity Governance, Granular RBAC, and Audit Log OffloadingEnterprise governance requires managing user memberships, access privileges, and audit logging across account resources. Cloudflare has modernized its identity architecture, transitioning from legacy, monolithic account roles (e.g., Super Administrator, Administrator, Read Only) to a granular Identity and Access Management (IAM) framework.Under modern Cloudflare IAM, permissions are structured into three composable objects: Permission Groups, Resource Groups, and User Groups. Permission Groups (/accounts/{account_id}/iam/permission_groups) define sets of permitted actions (such as editing DNS or reading log streams). Resource Groups (/accounts/{account_id}/iam/resource_groups) define the precise infrastructure scope to which those actions apply, isolating policies to specific zones, tunnels, or compute workers. User Groups (/accounts/{account_id}/iam/user_groups) aggregate account members.When inviting a user via /accounts/{account_id}/members, administrators can attach granular IAM policy blocks to the membership payload rather than relying on legacy flat role assignments. This prevents privilege escalation by ensuring that automated pipeline tokens and operational engineers receive strictly scoped permissions.Administrative auditing is governed by the Account Audit Logs API. Cloudflare has replaced its legacy v1 endpoint (/accounts/{account_id}/audit_logs) with the Audit Logs v2 interface (/accounts/{account_id}/logs/audit). Audit Logs v2 records state-changing administrative operations—capturing actor identity, source IP, affected resource IDs, old versus new JSON payloads, and interface channels. A key capability in v2 is the change history endpoint (/accounts/{account_id}/logs/audit/{id}/history), which exposes JSON patch diffs showing precisely what data changed within a given resource mutation.The Audit Logs REST API enforces a 30-day retention window for interactive queries, paginating results up to a maximum of 1,000 events per page. The API allows filtering on query parameters including since, before, action.type, actor.email, and direction. Because long-term compliance frameworks (SOC 2, ISO 27001, HIPAA) often require multi-year log retention, enterprise platforms should not rely exclusively on ad-hoc REST polling. Instead, operations teams should configure an automated Logpush job pointed at the audit_logs dataset. This automatically streams change logs to cold object storage (such as AWS S3 or Cloudflare R2) as actions occur.Wrangler contains no governance or audit-log features. Programmatic execution requires the REST API or the Terraform provider. Management calls operate at the account scope, requiring Account Settings: Edit, Account Settings: Read, or Audit Logs: Read permissions, and are subject to the standard 1,200 request per 5-minute rate limit.OperationMethod + EndpointScopeWrangler?NotesInvite Member with Scoped RBACPOST /accounts/{account_id}/membersAccount (Account Settings: Edit)NoDispatches an invitation attaching explicit IAM policy blocks, resource groups, or legacy role IDs.Provision IAM Resource GroupPOST /accounts/{account_id}/iam/resource_groupsAccount (Account Settings: Edit)NoCreates a granular resource boundary restricting member permissions to specific zones, tunnels, or scripts.Query Account Audit Logs (v2)GET /accounts/{account_id}/logs/auditAccount (Audit Logs: Read)NoQueries historical control-plane mutations within a 30-day window, supporting filtering by actor, action, and time.Fetch Audit Log Change HistoryGET /accounts/{account_id}/logs/audit/{id}/historyAccount (Audit Logs: Read)NoReturns explicit JSON patch diffs showing parameter-level changes for a recorded mutation event.Revoke Account MembershipDELETE /accounts/{account_id}/members/{member_id}Account (Account Settings: Edit)NoImmediately revokes an account member's administrative access, session tokens, and associated RBAC policies.Financial Operations Automation and Gated Commercial WorkflowsAutomating Cloudflare billing via code requires maintaining a strict distinction between capabilities accessible over the REST API and workflows that remain gated behind manual dashboard interfaces. Because billing mutations carry contractual, legal, and financial implications, Cloudflare restricts automated self-service around financial instruments and contractual agreements.The Billing REST API allows organizations to automate financial reporting, usage tracking, add-on subscription provisioning, and usage-based threshold limits. Automation pipelines can inspect billing profiles (/accounts/{account_id}/billing/profile), track real-time consumption across metered products (/accounts/{account_id}/billing/usage), and query historical invoice records and download itemized PDF receipts (/accounts/{account_id}/billing/invoices).Self-service add-ons—including Advanced Certificate Manager packs, dedicated egress IPs, Argo Smart Routing, and Workers Paid plan tiers—can be provisioned, updated, and cancelled programmatically via the Subscriptions API (/accounts/{account_id}/subscriptions). For services with metered usage models (such as Workers AI inference, Vectorize, and Workers KV), operators can manage credit limits, auto-topup triggers, and payment reload balances through /accounts/{account_id}/billing/topup/config to prevent production service lockouts.Several critical billing operations cannot be automated via public APIs and must be completed manually within the Cloudflare Dashboard:Payment Method Onboarding and Tokenization: Registering primary credit cards or linking corporate PayPal accounts cannot be automated over plain REST. In compliance with PCI-DSS requirements, adding a payment method requires client-side execution of Stripe Elements or PayPal iframes within an interactive browser to handle 3D Secure (3DS) authentication challenges. While the API exposes an endpoint to initiate a payment setup intent (POST /accounts/{account_id}/billing/payment_method/intent), completing verification requires an interactive user session.Enterprise Contract Adjustments: Enterprise agreements (Master Services Agreements, custom annual commits, committed bandwidth agreements, custom pricing pools) cannot be updated via self-service APIs. These adjustments require an executed sales order form processed through Cloudflare's enterprise billing desk.Tax Exemption Certifications: Uploading state, federal, or international VAT/tax exemption certificates must be processed manually through Cloudflare billing support.Currency and Entity Migrations: Changing an account's billing currency (e.g., from USD to EUR/GBP) or transferring ownership to a different corporate billing entity requires manual dashboard intervention and manual invoice reconciliation.Wrangler contains no billing capabilities. Modifying subscriptions or reading billing profiles requires high-privilege credentials—specifically an Account API Token with Billing: Edit or Billing: Read permissions, or a User API Key belonging to the account Super Administrator. Standard rate limits of 1,200 requests per 5 minutes apply across all billing endpoints.OperationMethod + EndpointScopeWrangler?NotesRead Billing ProfileGET /accounts/{account_id}/billing/profileAccount (Billing: Read)NoRetrieves the account's active billing profile, primary billing contact email, entity name, and payment status.Query Invoices & Download ReceiptsGET /accounts/{account_id}/billing/invoicesAccount (Billing: Read)NoRetrieves historical billing records, line-item totals, payment outcomes, and links to rendered PDF invoices.Provision Add-on SubscriptionPOST /accounts/{account_id}/subscriptionsAccount (Billing: Edit)NoProvisions paid platform add-ons (such as Argo Smart Routing, ACM, or Workers Paid) against the saved payment method.Configure Metered Auto Top-UpPUT /accounts/{account_id}/billing/topup/configAccount (Billing: Edit)NoConfigures wallet reload balance thresholds, top-up quantities, and notification triggers for metered products.Update Subscription AllocationsPUT /accounts/{account_id}/subscriptions/{subscription_id}Account (Billing: Edit)NoAdjusts subscription seat allocations, component quantities, or billing cycle renewals for active services.Establishing reliable, full-account control across Cloudflare requires orchestrating two distinct operational patterns: compute-centric services provide deep CLI and local development parity, while network-edge, security, and governance surfaces rely entirely on the REST control plane. By combining declarative configuration management with automated Logpush pipelines for long-term audit and log retention, engineering teams can build resilient infrastructure-as-code automation frameworks that operate cleanly within platform rate limits and security boundaries.COVERAGE JSONJSON[
  {
    "service": "Durable Objects",
    "management_ops_count": 5,
    "wrangler_coverage": "Partial (Class migrations and binding deployment via `wrangler deploy`; zero CLI commands for live object state inspection or data-plane manipulation)",
    "rest_endpoints_documented": [
      "GET /accounts/{account_id}/workers/durable_objects/namespaces",
      "POST /accounts/{account_id}/workers/durable_objects/namespaces",
      "GET /accounts/{account_id}/workers/durable_objects/namespaces/{namespace_id}/objects",
      "PUT /accounts/{account_id}/workers/scripts/{script_name}",
      "POST /client/v4/graphql"
    ],
    "permission_scopes": [
      "account: Workers Scripts: Read",
      "account: Workers Scripts: Edit",
      "account: Analytics: Read"
    ],
    "automatable_pct_estimate": 65,
    "source_url": "https://developers.cloudflare.com/durable-objects/api/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Workflows",
    "management_ops_count": 5,
    "wrangler_coverage": "Full (Native support for list, trigger, status inspect, pause, resume, terminate, send-event, and local dev simulation via `wrangler workflows`)",
    "rest_endpoints_documented": [
      "GET /accounts/{account_id}/workflows",
      "POST /accounts/{account_id}/workflows/{workflow_name}/instances",
      "POST /accounts/{account_id}/workflows/{workflow_name}/instances/batch",
      "GET /accounts/{account_id}/workflows/{workflow_name}/instances/{instance_id}",
      "PATCH /accounts/{account_id}/workflows/{workflow_name}/instances/{instance_id}/status"
    ],
    "permission_scopes": [
      "account: Workers Scripts: Read",
      "account: Workers Scripts: Edit"
    ],
    "automatable_pct_estimate": 100,
    "source_url": "https://developers.cloudflare.com/api/resources/workflows/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Cloudflare Tunnels",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed exclusively via `cloudflared` CLI binary or direct REST API / Terraform provider)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/cfd_tunnel",
      "GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/token",
      "PUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations",
      "POST /accounts/{account_id}/teamnet/virtual_networks",
      "POST /accounts/{account_id}/teamnet/routes",
      "GET /accounts/{account_id}/warp_connector"
    ],
    "permission_scopes": [
      "account: Cloudflare Tunnel: Edit",
      "account: Cloudflare Tunnel: Read",
      "zone: DNS: Edit"
    ],
    "automatable_pct_estimate": 95,
    "source_url": "https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Zero Trust Gateway + WARP",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed via REST API, Zero Trust dashboard, or Terraform provider; client uses local `warp-cli`)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/gateway/rules",
      "PUT /accounts/{account_id}/devices/policy/{policy_id}/exclude",
      "POST /accounts/{account_id}/devices/posture",
      "POST /accounts/{account_id}/devices/posture/integration",
      "DELETE /accounts/{account_id}/devices/{device_id}",
      "GET /accounts/{account_id}/devices"
    ],
    "permission_scopes": [
      "account: Zero Trust: Edit",
      "account: Zero Trust: Read",
      "account: Access: Apps and Policies: Edit"
    ],
    "automatable_pct_estimate": 90,
    "source_url": "https://developers.cloudflare.com/api/resources/zero_trust/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Long-tail Zone Services",
    "management_ops_count": 5,
    "wrangler_coverage": "None (All edge performance, security, and observability features managed via REST API or Terraform)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/load_balancers/pools",
      "PUT /zones/{zone_id}/bot_management",
      "POST /zones/{zone_id}/page_shield/policies",
      "POST /accounts/{account_id}/challenges/widgets",
      "POST /accounts/{account_id}/logpush/jobs",
      "POST /zones/{zone_id}/waiting_rooms",
      "POST /zones/{zone_id}/spectrum/apps",
      "POST /accounts/{account_id}/rum/site_info"
    ],
    "permission_scopes": [
      "account: Load Balancing: Edit",
      "zone: Load Balancing: Edit",
      "zone: Bot Management: Edit",
      "zone: Page Shield: Edit",
      "account: Turnstile: Edit",
      "account: Logs: Edit",
      "zone: Logs: Edit",
      "zone: Waiting Room: Edit",
      "zone: Spectrum: Edit",
      "account: Analytics: Edit"
    ],
    "automatable_pct_estimate": 95,
    "source_url": "https://developers.cloudflare.com/api/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Account/Org Management & Audit Logs",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed via REST API, Dashboard, or Terraform)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/members",
      "POST /accounts/{account_id}/iam/resource_groups",
      "GET /accounts/{account_id}/logs/audit",
      "GET /accounts/{account_id}/logs/audit/{id}/history",
      "DELETE /accounts/{account_id}/members/{member_id}",
      "GET /accounts/{account_id}/iam/permission_groups"
    ],
    "permission_scopes": [
      "account: Account Settings: Edit",
      "account: Account Settings: Read",
      "account: Audit Logs: Read"
    ],
    "automatable_pct_estimate": 95,
    "source_url": "https://developers.cloudflare.com/api/resources/accounts/subresources/logs/subresources/audit/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Billing API",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed via REST API for read/subscription operations; manual dashboard required for payment methods)",
    "rest_endpoints_documented": [
      "GET /accounts/{account_id}/billing/profile",
      "GET /accounts/{account_id}/billing/invoices",
      "POST /accounts/{account_id}/subscriptions",
      "PUT /accounts/{account_id}/billing/topup/config",
      "PUT /accounts/{account_id}/subscriptions/{subscription_id}",
      "GET /accounts/{account_id}/billing/usage"
    ],
    "permission_scopes": [
      "account: Billing: Read",
      "account: Billing: Edit"
    ],
    "automatable_pct_estimate": 45,
    "source_url": "https://developers.cloudflare.com/api/resources/billing/",
    "as_of": "2025-02-15"
  }
]


---

## COVERAGE JSON (recovered, fenced)

[
  {
    "service": "Durable Objects",
    "management_ops_count": 5,
    "wrangler_coverage": "Partial (Class migrations and binding deployment via `wrangler deploy`; zero CLI commands for live object state inspection or data-plane manipulation)",
    "rest_endpoints_documented": [
      "GET /accounts/{account_id}/workers/durable_objects/namespaces",
      "POST /accounts/{account_id}/workers/durable_objects/namespaces",
      "GET /accounts/{account_id}/workers/durable_objects/namespaces/{namespace_id}/objects",
      "PUT /accounts/{account_id}/workers/scripts/{script_name}",
      "POST /client/v4/graphql"
    ],
    "permission_scopes": [
      "account: Workers Scripts: Read",
      "account: Workers Scripts: Edit",
      "account: Analytics: Read"
    ],
    "automatable_pct_estimate": 65,
    "source_url": "https://developers.cloudflare.com/durable-objects/api/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Workflows",
    "management_ops_count": 5,
    "wrangler_coverage": "Full (Native support for list, trigger, status inspect, pause, resume, terminate, send-event, and local dev simulation via `wrangler workflows`)",
    "rest_endpoints_documented": [
      "GET /accounts/{account_id}/workflows",
      "POST /accounts/{account_id}/workflows/{workflow_name}/instances",
      "POST /accounts/{account_id}/workflows/{workflow_name}/instances/batch",
      "GET /accounts/{account_id}/workflows/{workflow_name}/instances/{instance_id}",
      "PATCH /accounts/{account_id}/workflows/{workflow_name}/instances/{instance_id}/status"
    ],
    "permission_scopes": [
      "account: Workers Scripts: Read",
      "account: Workers Scripts: Edit"
    ],
    "automatable_pct_estimate": 100,
    "source_url": "https://developers.cloudflare.com/api/resources/workflows/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Cloudflare Tunnels",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed exclusively via `cloudflared` CLI binary or direct REST API / Terraform provider)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/cfd_tunnel",
      "GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/token",
      "PUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations",
      "POST /accounts/{account_id}/teamnet/virtual_networks",
      "POST /accounts/{account_id}/teamnet/routes",
      "GET /accounts/{account_id}/warp_connector"
    ],
    "permission_scopes": [
      "account: Cloudflare Tunnel: Edit",
      "account: Cloudflare Tunnel: Read",
      "zone: DNS: Edit"
    ],
    "automatable_pct_estimate": 95,
    "source_url": "https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Zero Trust Gateway + WARP",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed via REST API, Zero Trust dashboard, or Terraform provider; client uses local `warp-cli`)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/gateway/rules",
      "PUT /accounts/{account_id}/devices/policy/{policy_id}/exclude",
      "POST /accounts/{account_id}/devices/posture",
      "POST /accounts/{account_id}/devices/posture/integration",
      "DELETE /accounts/{account_id}/devices/{device_id}",
      "GET /accounts/{account_id}/devices"
    ],
    "permission_scopes": [
      "account: Zero Trust: Edit",
      "account: Zero Trust: Read",
      "account: Access: Apps and Policies: Edit"
    ],
    "automatable_pct_estimate": 90,
    "source_url": "https://developers.cloudflare.com/api/resources/zero_trust/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Long-tail Zone Services",
    "management_ops_count": 5,
    "wrangler_coverage": "None (All edge performance, security, and observability features managed via REST API or Terraform)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/load_balancers/pools",
      "PUT /zones/{zone_id}/bot_management",
      "POST /zones/{zone_id}/page_shield/policies",
      "POST /accounts/{account_id}/challenges/widgets",
      "POST /accounts/{account_id}/logpush/jobs",
      "POST /zones/{zone_id}/waiting_rooms",
      "POST /zones/{zone_id}/spectrum/apps",
      "POST /accounts/{account_id}/rum/site_info"
    ],
    "permission_scopes": [
      "account: Load Balancing: Edit",
      "zone: Load Balancing: Edit",
      "zone: Bot Management: Edit",
      "zone: Page Shield: Edit",
      "account: Turnstile: Edit",
      "account: Logs: Edit",
      "zone: Logs: Edit",
      "zone: Waiting Room: Edit",
      "zone: Spectrum: Edit",
      "account: Analytics: Edit"
    ],
    "automatable_pct_estimate": 95,
    "source_url": "https://developers.cloudflare.com/api/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Account/Org Management & Audit Logs",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed via REST API, Dashboard, or Terraform)",
    "rest_endpoints_documented": [
      "POST /accounts/{account_id}/members",
      "POST /accounts/{account_id}/iam/resource_groups",
      "GET /accounts/{account_id}/logs/audit",
      "GET /accounts/{account_id}/logs/audit/{id}/history",
      "DELETE /accounts/{account_id}/members/{member_id}",
      "GET /accounts/{account_id}/iam/permission_groups"
    ],
    "permission_scopes": [
      "account: Account Settings: Edit",
      "account: Account Settings: Read",
      "account: Audit Logs: Read"
    ],
    "automatable_pct_estimate": 95,
    "source_url": "https://developers.cloudflare.com/api/resources/accounts/subresources/logs/subresources/audit/",
    "as_of": "2025-02-15"
  },
  {
    "service": "Billing API",
    "management_ops_count": 5,
    "wrangler_coverage": "None (Managed via REST API for read/subscription operations; manual dashboard required for payment methods)",
    "rest_endpoints_documented": [
      "GET /accounts/{account_id}/billing/profile",
      "GET /accounts/{account_id}/billing/invoices",
      "POST /accounts/{account_id}/subscriptions",
      "PUT /accounts/{account_id}/billing/topup/config",
      "PUT /accounts/{account_id}/subscriptions/{subscription_id}",
      "GET /accounts/{account_id}/billing/usage"
    ],
    "permission_scopes": [
      "account: Billing: Read",
      "account: Billing: Edit"
    ],
    "automatable_pct_estimate": 45,
    "source_url": "https://developers.cloudflare.com/api/resources/billing/",
    "as_of": "2025-02-15"
  }
]
