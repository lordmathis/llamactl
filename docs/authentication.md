# Authentication

Llamactl controls access with three mechanisms:

- **Management API Key** — authenticates the web UI and management API (creating/stopping instances, managing keys). Configured in the config file or via environment variables; see the [Configuration](configuration.md#authentication-configuration) guide.
- **Inference API Key** — authenticates OpenAI-compatible inference requests (`/v1/chat/completions`, `/v1/completions`, etc.). Created and managed via the web UI or management API and stored in the database.
- **OIDC / SSO Session** — optional browser login for the web UI through an OpenID Connect provider; see [OIDC / SSO Login](#oidc-sso-login) below.

The rest of this page covers **inference API keys** and their per-instance permissions, followed by **SSO setup**.

## Inference API Keys

### Permission Modes

When you create an inference key, you choose a permission mode:

- **Full Access** (`allow_all`): The key can use every instance with full lifecycle rights. This is the simplest option.
- **Per-Instance Access** (`per_instance`): You explicitly grant access to specific instances and choose an access level for each.

Management API keys bypass all permission checks entirely.

### Access Levels

For each instance granted to a per-instance key, you pick one of three access levels. A key can always **use a running instance** — the level only affects what happens when the instance is **stopped** at request time.

| Level | What the key can do |
|---|---|
| **Use running only** | Send requests only while the instance is already running. Never starts it. |
| **Can start on demand** | Auto-start the instance when needed, but won't disrupt others — fails if there's no free capacity. |
| **Can start and evict others** | Auto-start the instance and may evict other instances to free up room. |

!!! note "What &quot;evict&quot; means"
    The *Can start and evict others* level allows the key to **evict other instances** to make room for the one it's starting. It does **not** protect this instance from being evicted when a different key starts something else. Eviction always targets the least-recently-used instance.

Both flags default to enabled, so omitting them is equivalent to *Can start and evict others* (backward compatible with keys created before this feature).

### Behavior Reference

How a request is handled depends on the instance's state and the key's access level for it:

| Instance state | Access level | Result |
|---|---|---|
| Running | any | Request proxied normally |
| Stopped, free capacity | Use running only | `503 instance_not_running` |
| Stopped, free capacity | Can start (either) | Instance started, request proxied |
| Stopped, at capacity | Use running only | `503 instance_not_running` |
| Stopped, at capacity | Can start on demand | `503 max_instances_reached` |
| Stopped, at capacity | Can start and evict | Evicts LRU instance, starts, request proxied |

"Capacity" considers both the instance's [group limit](managing-instances.md#instance-groups) and the global `max_running_instances`. The two 503 responses use distinct error types so clients can tell them apart.

### Managing Keys

#### Via Web UI

1. Open the web UI and log in with a management API key
2. Navigate to **Settings → API Keys**
3. Click **Create API Key**
4. Configure the key:
    - **Name**: A descriptive name
    - **Permission Mode**: *Full Access* or *Per-Instance Access*
    - **Expiration**: Optional expiration date
    - **Instance Permissions** *(per-instance only)*: Check each instance and choose its access level
5. Click **Create**
6. **Copy the generated key** — it is only shown once

Expand an existing key in the list to review its per-instance access levels.

#### Via API

All key endpoints require a management API key (`<token>` below).

**Create a per-instance key with custom access levels:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "Production Key",
    "permission_mode": "per_instance",
    "permissions": [
      {"instance_id": 1, "can_start": true,  "can_evict": true},
      {"instance_id": 2, "can_start": true,  "can_evict": false},
      {"instance_id": 3, "can_start": false}
    ]
  }'
```

`can_start` and `can_evict` default to `true` when omitted. `instance_id` values correspond to the IDs returned by `GET /api/v1/instances`.

**List keys:**

```bash
curl http://localhost:8080/api/v1/auth/keys \
  -H "Authorization: Bearer <token>"
```

**View a key's instance permissions:**

```bash
curl http://localhost:8080/api/v1/auth/keys/{id}/permissions \
  -H "Authorization: Bearer <token>"
```

```json
[
  {"instance_id": 1, "instance_name": "my-llama-model", "can_start": true, "can_evict": true},
  {"instance_id": 2, "instance_name": "other-model",    "can_start": true, "can_evict": false}
]
```

**Delete a key:**

```bash
curl -X DELETE http://localhost:8080/api/v1/auth/keys/{id} \
  -H "Authorization: Bearer <token>"
```

### Use Cases

- **Production vs. development**: Give production keys *Can start and evict* so requests always succeed; give development keys *Can start on demand* so they never disrupt production workloads.
- **Read-only consumers**: Use *Use running only* for dashboards or monitoring tools that should reuse a warm instance but never spin one up.
- **Shared clusters**: Limit disruptive eviction rights to a small set of trusted keys so noisy neighbors can't evict each other's models.

## OIDC / SSO Login

The web UI can authenticate users through an OpenID Connect provider (Authentik, Keycloak, Authelia, etc.) instead of a management API key. OIDC login is enabled when both `issuer_url` and `client_id` are set; management API keys keep working alongside it, and inference endpoints always use API keys.

```yaml
auth:
  oidc:
    issuer_url: "https://authentik.example.com/application/o/llamactl/"  # IdP issuer URL
    client_id: "llamactl"                  # OAuth2 client ID registered with the IdP
    client_secret: "your-client-secret"    # OAuth2 client secret
    # redirect_url: ""                     # Optional; derived from the request when empty
    # scopes: [openid, profile, email]     # Default scopes
    # session_ttl: 12h                     # Session lifetime (default: 12h)
    # secure_cookie: true                  # Set false only for plain-HTTP LAN deployments
```

Environment variable equivalents (`LLAMACTL_AUTH_OIDC_*`) are listed in the [configuration reference](configuration.md#oidc-sso-login).

### Setting up the IdP client

1. Create a confidential OAuth2/OIDC client in your IdP (Authentik: provider type *OAuth2/OpenID Connect*; Keycloak: client with *Client authentication* enabled)
2. Set the redirect URI to `https://<your-llamactl-host>/api/v1/auth/oidc/callback` — under a subpath proxy, use the full external path
3. Configure the `auth.oidc` block above and restart llamactl; discovery failures abort startup with an actionable error

### Notes

- Keep `require_management_auth: true` when enabling OIDC — otherwise management endpoints stay unauthenticated and login only adds session support. llamactl logs a warning at startup for this combination.
- Behind a TLS-terminating reverse proxy, either forward `X-Forwarded-Proto`/`X-Forwarded-Host` or set `redirect_url` explicitly.
- The authorization code flow uses PKCE (S256) and a signed, one-shot CSRF state cookie. Sessions live in memory: a llamactl restart logs everyone out, and `secure_cookie: false` is only appropriate on trusted plain-HTTP LANs.
- API keys created while logged in via SSO are attributed to the user's OIDC subject (`sub`) instead of `system`.
