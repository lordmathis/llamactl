# API Keys

Llamactl uses two types of API keys to control access:

- **Management API Key** — authenticates the web UI and management API (creating/stopping instances, managing keys). See the [Configuration](configuration.md) guide for how to set these.
- **Inference API Key** — authenticates OpenAI-compatible inference requests (`/v1/chat/completions`, `/v1/completions`, etc.). These are created and managed via the web UI or management API and stored in the database.

This page focuses on **inference API keys** and how their per-instance permissions work.

## Permission Modes

When you create an inference key, you choose a permission mode:

- **Full Access** (`allow_all`): The key can use every instance with full lifecycle rights. This is the simplest option.
- **Per-Instance Access** (`per_instance`): You explicitly grant access to specific instances and choose an access level for each.

Management API keys bypass all permission checks entirely.

## Access Levels

For each instance granted to a per-instance key, you pick one of three access levels. A key can always **use a running instance** — the level only affects what happens when the instance is **stopped** at request time.

| Level | What the key can do |
|---|---|
| **Use running only** | Send requests only while the instance is already running. Never starts it. |
| **Can start on demand** | Auto-start the instance when needed, but won't disrupt others — fails if there's no free capacity. |
| **Can start and evict others** | Auto-start the instance and may evict other instances to free up room. |

!!! note "What &quot;evict&quot; means"
    The *Can start and evict others* level allows the key to **evict other instances** to make room for the one it's starting. It does **not** protect this instance from being evicted when a different key starts something else. Eviction always targets the least-recently-used instance.

Both flags default to enabled, so omitting them is equivalent to *Can start and evict others* (backward compatible with keys created before this feature).

## Behavior Reference

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

## Managing Keys

### Via Web UI

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

### Via API

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

## Use Cases

- **Production vs. development**: Give production keys *Can start and evict* so requests always succeed; give development keys *Can start on demand* so they never disrupt production workloads.
- **Read-only consumers**: Use *Use running only* for dashboards or monitoring tools that should reuse a warm instance but never spin one up.
- **Shared clusters**: Limit disruptive eviction rights to a small set of trusted keys so noisy neighbors can't evict each other's models.
