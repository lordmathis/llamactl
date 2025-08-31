# Quick Start

This guide will help you get Llamactl up and running in just a few minutes.

## Step 1: Start Llamactl

Start the Llamactl server:

```bash
llamactl
```

By default, Llamactl will start on `http://localhost:8080`.

## Step 2: Access the Web UI

Open your web browser and navigate to:

```
http://localhost:8080
```

You should see the Llamactl web interface.

## Step 3: Create Your First Instance

1. Click the "Add Instance" button
2. Fill in the instance configuration:
   - **Name**: Give your instance a descriptive name
   - **Model Path**: Path to your Llama.cpp model file
   - **Port**: Port for the instance to run on
   - **Additional Options**: Any extra Llama.cpp parameters

3. Click "Create Instance"

## Step 4: Start Your Instance

Once created, you can:

- **Start** the instance by clicking the start button
- **Monitor** its status in real-time
- **View logs** by clicking the logs button
- **Stop** the instance when needed

## Example Configuration

Here's a basic example configuration for a Llama 2 model:

```json
{
  "name": "llama2-7b",
  "model_path": "/path/to/llama-2-7b-chat.gguf",
  "port": 8081,
  "options": {
    "threads": 4,
    "context_size": 2048
  }
}
```

## Using the API

You can also manage instances via the REST API:

```bash
# List all instances
curl http://localhost:8080/api/instances

# Create a new instance
curl -X POST http://localhost:8080/api/instances \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-model",
    "model_path": "/path/to/model.gguf",
    "port": 8081
  }'

# Start an instance
curl -X POST http://localhost:8080/api/instances/my-model/start
```

## Next Steps

- Learn more about the [Web UI](../user-guide/web-ui.md)
- Explore the [API Reference](../user-guide/api-reference.md)
- Configure advanced settings in the [Configuration](configuration.md) guide
