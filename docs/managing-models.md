# Managing Models

Llamactl provides built-in model management capabilities for **llama.cpp (GGUF) models**, allowing you to download models directly from HuggingFace without manually managing files. This feature replicates the `--hf` behavior from llama.cpp.

## Overview

The model downloader:
- Downloads GGUF models directly from HuggingFace repositories
- Caches models in a local directory for reuse
- Supports split files (models split into multiple parts)
- Downloads additional files like mmproj (for multimodal models) and preset.ini
- Tracks download progress and manages jobs
- Works with both public and private HuggingFace repositories
- **Only supports llama.cpp backend**

## Cache Directory

Downloaded models are cached in the following default locations:

- **Linux**: `~/.cache/llama.cpp/` (or `$XDG_CACHE_HOME/llama.cpp/` if XDG_CACHE_HOME is set)
- **macOS**: `~/Library/Caches/llama.cpp/`
- **Windows**: `%LOCALAPPDATA%\llama.cpp\cache\`

The cache directory can be customized using:
- Configuration file: `backends.llama_cpp.cache_dir`
- Environment variable: `LLAMACTL_LLAMACPP_CACHE_DIR`
- Environment variable: `LLAMA_CACHE` (standard llama.cpp convention)

## Downloading Models

### Using the Web UI

1. Open the Llamactl web interface
2. Navigate to the **Models** tab
3. Click **Download Model**
4. Enter the model identifier in the format: `org/model-name:tag`
   - Example: `bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M`
   - The `:tag` part is optional - if omitted, `latest` is used
5. Click **Start Download**
6. Monitor the download progress in the UI

The Models tab shows:
- **Downloaded Bytes**: Progress of current file download
- **Current File**: Name of the file being downloaded
- **Status**: Current job status (queued, downloading, completed, failed, cancelled)

### Using the API

Download a model programmatically:

```bash
curl -X POST http://localhost:8080/api/v1/backends/llama-cpp/models/download \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "repo": "bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M"
  }'
```

Response:
```json
{
  "job_id": "a1b2c3d4e5f6g7h8",
  "repo": "bartowski/Llama-3.2-3B-Instruct-GGUF",
  "tag": "Q4_K_M"
}
```

### Model Format

Models are specified in the format: `org/model-name` or `org/model-name:tag`

- **org**: The HuggingFace organization or user (e.g., `bartowski`, `microsoft`)
- **model-name**: The model repository name
- **tag**: (Optional) The specific model variant or quantization. If omitted, defaults to `latest`

Examples:
- `bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M` - Specific quantization
- `bartowski/Llama-3.2-3B-Instruct-GGUF` - Uses `latest` tag

## What Gets Downloaded

When you download a model, Llamactl fetches:

1. **Manifest**: Contains metadata about the model files
2. **GGUF File(s)**: The main model file(s) - may be split into multiple parts
3. **Split Files**: If the model is split (e.g., `model-00001-of-00003.gguf`), all parts are downloaded
4. **MMProj File**: (Optional) For multimodal models that support vision
5. **Preset File**: (Optional) `preset.ini` with recommended inference settings

Files are downloaded with ETags to support efficient caching and validation.

## Private Models

To download private HuggingFace models, set the `HF_TOKEN` environment variable:

```bash
export HF_TOKEN="hf_your_token_here"
llamactl
```

The token will be used to authenticate requests to HuggingFace.

## Listing Cached Models

### Using the Web UI

Navigate to the **Models** tab to see all cached models with their sizes and file counts.

### Using the API

List all cached models:

```bash
curl http://localhost:8080/api/v1/backends/llama-cpp/models \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY"
```

Response:
```json
[
  {
    "repo": "bartowski/Llama-3.2-3B-Instruct-GGUF",
    "tag": "Q4_K_M",
    "files": [
      {
        "name": "model.gguf",
        "size": 2147483648,
        "modified": 1704067200
      }
    ],
    "total_size": 2147483648,
    "file_count": 1
  }
]
```

## Deleting Cached Models

### Using the Web UI

1. Navigate to the **Models** tab
2. Find the model you want to delete
3. Click the **Delete** button
4. Confirm the deletion

### Using the API

Delete a cached model:

```bash
curl -X DELETE "http://localhost:8080/api/v1/backends/llama-cpp/models?repo=bartowski/Llama-3.2-3B-Instruct-GGUF&tag=Q4_K_M" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY"
```

The `tag` parameter is optional. If omitted, all versions of the model will be deleted.

## Using Downloaded Models

Once a model is downloaded, you need to provide the **full path to the cached file** when creating llama.cpp instances.

### Finding the Cached File Path

Downloaded models are cached with the naming pattern: `{org}_{model}_{filename}`

Example:
```
Repo: bartowski/Llama-3.2-3B-Instruct-GGUF
Tag: Q4_K_M
File: model-Q4_K_M.gguf

Cached as: ~/.cache/llama.cpp/bartowski_Llama-3.2-3B-Instruct-GGUF_model-Q4_K_M.gguf
```

You can find the exact path by:
1. Using the **Models** tab in the Web UI (shows file paths)
2. Listing models via the API (includes file paths in response)

### Creating an Instance with a Downloaded Model

**Via the Web UI:**

1. Click **Add Instance**
2. Select **llama.cpp** as the backend type
3. In the **Model** field, enter the full path to the cached file:
   - Example: `~/.cache/llama.cpp/bartowski_Llama-3.2-3B-Instruct-GGUF_model-Q4_K_M.gguf`
   - Or use the absolute path: `/home/user/.cache/llama.cpp/bartowski_Llama-3.2-3B-Instruct-GGUF_model-Q4_K_M.gguf`

**Via the API:**

```bash
curl -X POST http://localhost:8080/api/v1/instances/my-llama-instance \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "backend_type": "llama_cpp",
    "backend_options": {
      "model": "~/.cache/llama.cpp/bartowski_Llama-3.2-3B-Instruct-GGUF_model-Q4_K_M.gguf"
    }
  }'
```

### Alternative: Using llama.cpp's Built-in HuggingFace Support

Instead of using the model downloader, you can let llama.cpp download models directly using its built-in HuggingFace support:

```bash
curl -X POST http://localhost:8080/api/v1/instances/my-llama-instance \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "backend_type": "llama_cpp",
    "backend_options": {
      "hf_repo": "bartowski/Llama-3.2-3B-Instruct-GGUF",
      "hf_file": "model-Q4_K_M.gguf"
    }
  }'
```
