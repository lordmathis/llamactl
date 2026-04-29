# Managing Models

Llamactl provides built-in model management capabilities for downloading models directly from HuggingFace without manually managing files. It supports both **GGUF** models (for llama.cpp) and **Safetensors** models (for vLLM and MLX backends).

## Overview

The model downloader:
- Downloads models directly from HuggingFace repositories in two formats: **GGUF** and **Safetensors**
- Caches models in a local directory using HuggingFace Hub-compatible layout for reuse
- Supports split files (models split into multiple parts) for both formats
- Downloads additional files like mmproj (for multimodal GGUF models) and preset.ini
- Downloads tokenizer, config, and all weight files for safetensors models
- Tracks download progress and manages jobs
- Works with both public and private HuggingFace repositories
- Supports resume of interrupted downloads

## Cache Directory

Downloaded models are cached in the following default locations:

- **Linux**: `~/.cache/llama.cpp/` (or `$XDG_CACHE_HOME/llama.cpp/` if XDG_CACHE_HOME is set)
- **macOS**: `~/Library/Caches/llama.cpp/`
- **Windows**: `%LOCALAPPDATA%\llama.cpp\cache\`

The cache directory can be customized using:
- Configuration file: `backends.llama_cpp.cache_dir`
- Environment variable: `LLAMACTL_LLAMACPP_CACHE_DIR`
- Environment variable: `LLAMA_CACHE` (standard llama.cpp convention)

## Supported Formats

Llamactl supports downloading models in two formats:

### GGUF (for llama.cpp)

GGUF is the native format for llama.cpp. GGUF models are pre-quantized and contain everything needed for inference in a single file (or a set of split files).

- Use with the **llama.cpp** backend
- Specify a quantization tag to select a specific variant (e.g., `Q4_K_M`, `Q8_0`)
- Example repos: `bartowski/Llama-3.2-3B-Instruct-GGUF`, `TheBloke/Mistral-7B-Instruct-v0.2-GGUF`

### Safetensors (for vLLM and MLX)

Safetensors is the standard model format used by HuggingFace. Downloads include all weight files, tokenizer files, config files, and other mandatory files needed to load the model.

- Use with **vLLM** or **MLX** backends
- Downloads the full model directory (all safetensors weight shards, tokenizer, config, etc.)
- Falls back to `.bin` files if no safetensors files are found in the repo
- Example repos: `meta-llama/Llama-3.2-3B`, `mistralai/Mistral-7B-Instruct-v0.3`

## Downloading Models

### Using the Web UI

1. Open the Llamactl web interface
2. Navigate to the **Models** tab
3. Click **Download Model**
4. Select the **Format**: **GGUF** or **Safetensors**
5. Enter the model identifier:
   - **GGUF**: `org/model-name:tag` (e.g., `bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M`)
   - **Safetensors**: `org/model-name` (e.g., `meta-llama/Llama-3.2-3B`)
   - The `:tag` part is optional - if omitted, the default branch is used
6. Click **Start Download**
7. Monitor the download progress in the UI

The Models tab shows:
- **Downloaded Bytes**: Progress of current file download
- **Current File**: Name of the file being downloaded
- **Status**: Current job status (queued, downloading, completed, failed, cancelled)

### Using the API

Download a GGUF model:

```bash
curl -X POST http://localhost:8080/api/v1/models/download \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "repo": "bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M",
    "format": "gguf"
  }'
```

Download a safetensors model:

```bash
curl -X POST http://localhost:8080/api/v1/models/download \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "repo": "meta-llama/Llama-3.2-3B",
    "format": "safetensors"
  }'
```

Response:
```json
{
  "job_id": "a1b2c3d4e5f6g7h8",
  "repo": "meta-llama/Llama-3.2-3B",
  "tag": "main"
}
```

The `format` field accepts `"gguf"` (default) or `"safetensors"`.

### Model Identifier Format

Models are specified in the format: `org/model-name` or `org/model-name:tag`

- **org**: The HuggingFace organization or user (e.g., `bartowski`, `meta-llama`)
- **model-name**: The model repository name
- **tag**: (Optional) The branch, tag, or specific quantization variant. If omitted, the default branch is used

Examples:
- `bartowski/Llama-3.2-3B-Instruct-GGUF:Q4_K_M` - Specific GGUF quantization
- `meta-llama/Llama-3.2-3B` - Safetensors model, default branch
- `meta-llama/Llama-3.2-3B:main` - Safetensors model, explicit branch

## What Gets Downloaded

What gets downloaded depends on the selected format:

### GGUF Downloads

1. **GGUF File(s)**: The main model file(s) - may be split into multiple parts
2. **Split Files**: If the model is split (e.g., `model-00001-of-00003.gguf`), all parts are downloaded
3. **MMProj File**: (Optional) For multimodal models that support vision
4. **Preset File**: (Optional) `preset.ini` with recommended inference settings
5. **Jinja Files**: (Optional) Chat template files

### Safetensors Downloads

1. **Weight Files**: All `.safetensors` files (or `.bin` files as fallback)
2. **Tokenizer Files**: `tokenizer.json`, `tokenizer_config.json`, `tokenizers/*.json`, etc.
3. **Config Files**: `config.json` and other `.json` configuration files
4. **Other Mandatory Files**: `.txt`, `.model`, `.py`, and `.jinja` files

All files are stored using HuggingFace Hub-compatible cache layout with blob storage and symlinks.

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
curl http://localhost:8080/api/v1/models \
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
curl -X DELETE "http://localhost:8080/api/v1/models?repo=bartowski/Llama-3.2-3B-Instruct-GGUF&tag=Q4_K_M" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY"
```

The `tag` parameter is optional. If omitted, all versions of the model will be deleted.

## Using Downloaded Models

Once a model is downloaded, you can use it when creating instances. The way you reference the model depends on the format.

### GGUF Models (llama.cpp)

For GGUF models, provide the **full path to the cached file** when creating llama.cpp instances.

**Creating an instance with a downloaded GGUF model:**

**Via the Web UI:**

1. Click **Add Instance**
2. Select **llama.cpp** as the backend type
3. In the **Model** field, enter the full path to the cached file

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

### Safetensors Models (vLLM / MLX)

For safetensors models, simply use the **HuggingFace repo identifier** (e.g., `meta-llama/Llama-3.2-3B`) when creating instances. Both vLLM and MLX will automatically resolve the repo to the locally cached snapshot.

**vLLM backend:**

```bash
curl -X POST http://localhost:8080/api/v1/instances/my-vllm-instance \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "backend_type": "vllm",
    "backend_options": {
      "model": "meta-llama/Llama-3.2-3B"
    }
  }'
```

**MLX backend:**

```bash
curl -X POST http://localhost:8080/api/v1/instances/my-mlx-instance \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  -d '{
    "backend_type": "mlx_lm",
    "backend_options": {
      "model": "meta-llama/Llama-3.2-3B"
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
