# Custom Backends

Custom backends provide support for inference servers without native llamactl integration. Any server can be managed (launched, health-checked, proxied, and evicted) by defining it as a named entry in the config file.

## How It Works

A custom backend is a named entry under `backends.custom` in the config file; instances reference the entry by name, and llamactl runs the configured command with the instance's arguments. The available fields are documented in [Backend Configuration](configuration.md#custom-backends).

The managed server must meet three requirements:

- It starts an HTTP server that listens on a TCP port.
- It serves inference endpoints under `/v1/`. All `POST /v1/*` requests are proxied to the instance, so OpenAI-style (`/v1/chat/completions`, `/v1/responses`) and Anthropic-style (`/v1/messages`) conventions all work.
- It exposes a health endpoint that returns a 2xx response when the server is ready (see [Health Checks](#health-checks)).

The server learns its port through the `{port}` placeholder in its arguments — llamactl has no other way to communicate it.

On multi-node setups, the command must exist on the node that runs the instance.

## Creating an Instance

Instances of a custom backend are created like any other instance, with `backend_type: "custom"`; see [Managing Instances](managing-instances.md) for the API examples and instance options. In the web UI, each configured name appears in the backend dropdown.

The launched command receives arguments from two places: the entry's `args` first, then the instance's `args`. Both support two placeholders, substituted when the instance starts:

- `{port}` — the port assigned to the instance. The combined arguments must contain it somewhere, otherwise the server cannot learn its port and instance creation is rejected.
- `{model}` — the model identifier of the instance. Optional; if the arguments use it, the instance must have a model set.

Servers that download models or run long setup on first start may exceed the on-demand start timeout (`instances.on_demand_start_timeout`). Raise the timeout or pre-warm the server's model cache by running the command once manually.

## Health Checks

After launching the command, llamactl polls the entry's `health_path` until it returns a 2xx response, and only then marks the instance ready.
