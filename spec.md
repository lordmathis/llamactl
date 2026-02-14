# Implementation Spec: preset.ini Editor for llama.cpp Router Mode

## Overview

Add **optional** support for creating and editing `preset.ini` files within the instance dialog. This enables users who want to use llama.cpp router mode to configure multiple models through a simple text editor with autocomplete suggestions. Router mode and preset.ini are completely optional features - instances work normally without them.

## Goals

1. Allow users to **optionally** create preset.ini configurations for llama.cpp router mode
2. Provide autocomplete suggestions for field names based on existing schema
3. Store preset.ini content per-instance (not globally)
4. Support import/export of preset.ini files
5. Make the feature discoverable but not mandatory - instances work fine without it

## Non-Goals

- Automatic generation of presets from existing instances
- Shared/global preset library
- Visual form builder for presets (keep it as a text editor)
- Validation of INI syntax (let llama.cpp handle this)

## Architecture

### Data Flow

```
User edits INI in UI
  ↓
Stored as string in instance options (preset_ini field)
  ↓
Sent to backend on instance create/update
  ↓
Backend writes to {instance_dir}/preset.ini
  ↓
Backend sets models_preset option to point to created preset.ini
  ↓
llama-server starts with --models-preset {instance_dir}/preset.ini
  ↓
llama-server reads and uses preset.ini for router mode
```

### How preset.ini is Passed to llama-server

**Note: Router mode and preset.ini are completely optional.** Users can run llama.cpp instances without router mode.

For users who want to use router mode with multiple models, the `preset.ini` file is passed to llama-server via the `--models-preset` command-line option. Users have three options:

1. **No router mode** (Default): Don't create preset.ini and don't set models_preset. Instance runs normally with a single model.

2. **Auto-generated preset.ini**: Create the preset.ini content in the Preset tab. The backend will:
   - Write the content to `{instance_dir}/preset.ini`
   - Automatically set `models_preset` backend option to this path
   - Pass `--models-preset {instance_dir}/preset.ini` to llama-server

3. **Custom preset.ini path**: Manually specify a path in the `models_preset` backend option (moved to Basic Backend Options). This allows users to:
   - Use a preset.ini file located elsewhere on the filesystem
   - Share preset.ini files between multiple instances
   - Skip the Preset tab entirely

### Relationship Between preset_ini and models_preset

**Both fields are completely optional.** Router mode is an optional feature for advanced use cases.

These are two separate but related fields in the instance options:

- **`preset_ini`** (string, instance-level option): The **content** of the preset.ini file
  - **Completely optional** - can be left empty/undefined
  - Edited in the Preset tab UI
  - Stored in the database as part of instance options
  - Written to `{instance_dir}/preset.ini` when instance is created/updated (if provided)
  - Empty by default - users only fill this if they want router mode

- **`models_preset`** (string, backend option): The **path** to the preset.ini file
  - **Completely optional** - can be left empty/undefined
  - Set in Basic Backend Options
  - Only passed to llama-server as `--models-preset` flag if a value is set
  - Auto-set to `{instance_dir}/preset.ini` if `preset_ini` content is provided and `models_preset` is empty
  - Can be manually set to any path for custom preset.ini files
  - Already exists in `pkg/backends/llama.go` (line 207) and `webui/src/schemas/backends/llamacpp.ts` (line 187)

**Priority/Precedence:**
1. If neither is provided → **llama-server runs normally without router mode** (default behavior)
2. If user sets `models_preset` manually → use that path (even if `preset_ini` content exists)
3. If user provides `preset_ini` content but no `models_preset` → auto-set `models_preset` to generated file path

### Storage Location

- **Per-instance**: Each instance has its own preset.ini in its working directory
- **Database**: preset.ini content stored as text field in instance options
- **File system**: Written to disk when instance is created

## Implementation

### Phase 1: Frontend Implementation

#### Data Model

**File:** `webui/src/types/instance.ts`

Add `preset_ini` field to instance options:

```typescript
export interface CreateInstanceOptions {
  // ... existing fields
  preset_ini?: string // Raw INI file content
}
```

**File:** `webui/src/schemas/instanceOptions.ts`

Add to schema:

```typescript
export const CreateInstanceOptionsSchema = z.object({
  // ... existing fields
  preset_ini: z.string().optional(),
})
```

**Tests:** Verify preset_ini field is optional in schema validation.

#### Move models_preset to Basic Backend Options

**File:** `webui/src/lib/zodFormUtils.ts`

Add `models_preset` to the `basicLlamaCppFieldsConfig` object (currently at lines 17-42):

```typescript
const basicLlamaCppFieldsConfig: Record<string, {
  label: string
  description?: string
  placeholder?: string
}> = {
  model: {
    label: 'Model Path',
    placeholder: '/path/to/model.gguf',
    description: 'Path to the model file'
  },
  hf_repo: {
    label: 'Hugging Face Repository',
    placeholder: 'microsoft/DialoGPT-medium',
    description: 'Hugging Face model repository'
  },
  // Remove hf_file from basic options
  gpu_layers: {
    label: 'GPU Layers',
    placeholder: '0',
    description: 'Number of layers to offload to GPU'
  },
  models_preset: {
    label: 'Models Preset Path',
    placeholder: '/path/to/preset.ini',
    description: 'Optional: Path to preset.ini for router mode. Leave empty for normal operation, or use Preset tab to auto-generate'
  }
}
```

This moves `models_preset` from the Advanced tab to the Basic Backend Options section in the Backend tab, making it more accessible to users who want to use router mode.

**Dynamic State Indication:**

The `models_preset` field should provide visual feedback about its current state:

1. **Empty + No preset_ini content** → Show: "No router mode (leave empty)"
2. **Empty + preset_ini has content** → Show: "Will auto-set to: {instance_dir}/preset.ini" (info/help text below field)
3. **User-provided value** → Show: "Custom path: {value}" or just display the value normally

Implementation approach in `BackendFormField` component:
- Access `formData.preset_ini` to check if preset content exists
- If `models_preset` is empty AND `preset_ini` has content, show help text:
  - "This will be auto-set to the preset.ini created in Preset tab"
- If `models_preset` has a value, it's a custom path (normal display)
- Consider adding a small badge/indicator (e.g., "Auto" badge) when it will be auto-generated

**Tests:** Verify models_preset shows correct state (empty/auto/custom) based on preset_ini content.

#### Field Suggestions

**File:** `webui/src/lib/llamaFieldSuggestions.ts` (new)

```typescript
import {
  getAllLlamaCppFieldKeys,
  getLlamaCppFieldType
} from '@/schemas/instanceOptions'

export interface FieldSuggestion {
  name: string // kebab-case for INI
  type: 'text' | 'number' | 'boolean' | 'array'
}

function snakeToKebab(snake: string): string {
  return snake.replace(/_/g, '-')
}

export function getLlamaFieldSuggestions(input: string): FieldSuggestion[] {
  const allFields = getAllLlamaCppFieldKeys()
  const query = input.toLowerCase().trim()

  const suggestions: FieldSuggestion[] = allFields
    .filter(field => field !== 'extra_args') // Don't suggest extra_args in INI
    .map(field => ({
      name: snakeToKebab(field),
      type: getLlamaCppFieldType(field)
    }))

  if (!query) return suggestions.slice(0, 20)

  return suggestions.filter(s => s.name.includes(query))
}
```

**Tests:** Verify field suggestions convert snake_case to kebab-case and exclude extra_args.

#### INI Editor Component

**File:** `webui/src/components/instance/IniEditor.tsx` (new)

Features:
- Textarea with monospace font
- Autocomplete dropdown triggered on typing
- Keyboard navigation (Arrow keys, Enter/Tab to accept, Escape to close)
- Insert suggestion at cursor position

Implementation details:
- Monitor cursor position and current line
- Show suggestions when typing field names (before `=`)
- Don't show suggestions in section headers `[...]` or comments `;`
- Filter suggestions based on partial input
- Highlight selected suggestion
- Apply suggestion by replacing current field name

Key methods:
- `updateSuggestions(text, cursorPos)` - analyze current context and show/hide suggestions
- `applySuggestion(suggestion)` - insert suggestion at cursor
- `handleKeyDown(e)` - keyboard navigation for suggestions

**Tests:** Verify autocomplete suggestions appear for field names but not in comments/sections.

#### Preset Tab

**File:** `webui/src/components/instance/PresetTab.tsx` (new)

UI Structure:
```
┌─────────────────────────────────────────┐
│ Preset Tab                              │
├─────────────────────────────────────────┤
│ Title: "Model Presets (preset.ini)"    │
│ Description: "Optional: Configure       │
│              multiple models for router │
│              mode. Leave empty to run   │
│              without router mode. This  │
│              will auto-set models-      │
│              preset option unless you   │
│              specify a custom path."    │
│                                         │
│ [Import] [Export]                       │
│                                         │
│ ┌─────────────────────────────────┐   │
│ │                                 │   │
│ │  <IniEditor>                    │   │
│ │                                 │   │
│ │                                 │   │
│ └─────────────────────────────────┘   │
│                                        │
└─────────────────────────────────────────┘
```

Features:
- Import button: Opens file picker for .ini files
- Export button: Downloads current content as preset.ini
- Info text explaining relationship with models_preset option

Props:
```typescript
interface PresetTabProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
}
```

#### Instance Dialog Integration

**File:** `webui/src/components/InstanceDialog.tsx`

Changes:
1. Add "Preset" tab to TabsList (only show for llama.cpp instances)
2. Add TabsContent for preset tab
3. Update tabs array to include "preset"

```typescript
const tabs = ["general", "backend", "advanced", "preset"]

// In TabsList
{formData.backend_type === BackendType.LLAMA_CPP && (
  <TabsTrigger value="preset">Preset</TabsTrigger>
)}

// In Tabs content area
<TabsContent value="preset" className="h-full">
  <PresetTab
    formData={formData}
    onChange={handleFieldChange}
  />
</TabsContent>
```

#### Enhanced BackendFormField for models_preset

**File:** `webui/src/components/BackendFormField.tsx`

Add special handling for the `models_preset` field to show dynamic state:

```typescript
// Add to BackendFormField component
const BackendFormField: React.FC<BackendFormFieldProps> = ({
  fieldKey,
  value,
  onChange,
  // Add new prop to access full form data
  formData
}) => {
  // Special handling for models_preset field
  if (fieldKey === 'models_preset') {
    const hasPresetContent = formData?.preset_ini && formData.preset_ini.trim().length > 0
    const isCustomPath = value && value.toString().trim().length > 0

    let helpText = 'Optional: Path to preset.ini for router mode'
    let badge = null

    if (!isCustomPath && hasPresetContent) {
      helpText = 'Will be auto-set to the preset.ini created in Preset tab'
      badge = <Badge variant="secondary" className="ml-2">Auto</Badge>
    } else if (isCustomPath) {
      badge = <Badge variant="outline" className="ml-2">Custom</Badge>
    }

    return (
      <div className="space-y-2">
        <div className="flex items-center">
          <Label htmlFor={fieldKey}>Models Preset Path</Label>
          {badge}
        </div>
        <Input
          id={fieldKey}
          value={value?.toString() || ''}
          onChange={(e) => onChange(fieldKey, e.target.value)}
          placeholder="/path/to/preset.ini"
        />
        <p className="text-sm text-muted-foreground">{helpText}</p>
      </div>
    )
  }

  // ... rest of the component for other fields
}
```

**Changes to BackendTab:**
- Pass `formData` prop to `BackendFormField` for models_preset field
- This allows the field to check if preset_ini has content

```typescript
<BackendFormField
  key={fieldKey}
  fieldKey={fieldKey}
  value={(formData.backend_options as Record<string, unknown>)?.[fieldKey]}
  onChange={onBackendFieldChange}
  formData={formData} // Add this
/>
```

**Tests:** Verify BackendFormField displays correct badge and help text for all three states.

### Phase 2: Backend Implementation

#### Data Model

**File:** `pkg/instance/options.go`

Add field to Options struct:

```go
type Options struct {
    // ... existing fields
    PresetIni *string `json:"preset_ini,omitempty"`
}
```

**Tests:** Verify preset_ini serializes to/from JSON correctly.

#### Write preset.ini on Instance Creation and Set models_preset

**File:** `pkg/instance/instance.go`

Add helper function to write preset.ini and automatically set models_preset:

```go
// writePresetIni writes the preset.ini file if provided in options and updates models_preset
func writePresetIni(name string, opts *Options, logsDir string) error {
    if opts == nil || opts.PresetIni == nil || *opts.PresetIni == "" {
        return nil // No preset.ini to write
    }

    // Create instance directory alongside logs directory
    instanceDir := filepath.Join(filepath.Dir(logsDir), name)
    if err := os.MkdirAll(instanceDir, 0755); err != nil {
        return fmt.Errorf("failed to create instance directory: %w", err)
    }

    presetPath := filepath.Join(instanceDir, "preset.ini")
    if err := os.WriteFile(presetPath, []byte(*opts.PresetIni), 0644); err != nil {
        return fmt.Errorf("failed to write preset.ini: %w", err)
    }

    // Automatically set models_preset to point to the created preset.ini
    // Only set if user hasn't already specified a custom path
    if llamaOpts, ok := opts.BackendOptions.(*backends.LlamaCppOptions); ok {
        if llamaOpts.ModelsPreset == "" {
            llamaOpts.ModelsPreset = presetPath
            log.Printf("Set models_preset to %s for instance %s", presetPath, name)
        }
    }

    log.Printf("Wrote preset.ini for instance %s at %s", name, presetPath)
    return nil
}
```

In `New()` function, after logger creation:

```go
func New(name string, globalConfig *config.AppConfig, opts *Options, onStatusChange func(oldStatus, newStatus Status)) *Instance {
    // ... existing code ...

    // Only create logger, proxy, and process for local instances
    if !instance.IsRemote() {
        logRotationConfig := &LogRotationConfig{
            Enabled:  globalInstanceSettings.LogRotationEnabled,
            MaxSize:  globalInstanceSettings.LogRotationMaxSize,
            Compress: globalInstanceSettings.LogRotationCompress,
        }
        instance.logger = newLogger(
            name,
            globalInstanceSettings.LogsDir,
            logRotationConfig,
        )
        instance.process = newProcess(instance)

        // Write preset.ini if provided and set models_preset
        if err := writePresetIni(name, opts, globalInstanceSettings.LogsDir); err != nil {
            log.Printf("Warning: Failed to write preset.ini for instance %s: %v", name, err)
            // Don't fail instance creation, just log the warning
        }
    }

    return instance
}
```

**Important Notes:**
- If user has already set a custom `models_preset` path in backend options, it takes precedence over the auto-generated path
- The auto-generated preset.ini is written to `{instance_dir}/preset.ini` where `instance_dir` is `{logs_parent_dir}/{instance_name}`
- The `models_preset` field (defined in `pkg/backends/llama.go` at line 207) is automatically converted to `--models-preset` CLI flag by the builder

**Tests:** `pkg/instance/instance_test.go`
- Instance with preset_ini creates file and auto-sets models_preset to generated path
- Instance with custom models_preset doesn't overwrite it even when preset_ini exists
- Empty preset_ini doesn't create file

**Documentation:** Update user documentation to explain:
- Router mode configuration via preset.ini tab
- How preset.ini is automatically used when created
- How to override with custom models_preset path


## Edge Cases

### Empty Preset
- Empty string or whitespace-only → don't write preset.ini file
- Null/undefined → no preset.ini (default behavior)
- If preset_ini is empty but models_preset is set → use the models_preset path as-is

### Invalid INI Syntax
- Don't validate on frontend (let llama-server handle it)
- If llama-server fails to start, error will appear in logs
- User can fix via instance edit → preset tab

### Instance Directory
- Create directory if doesn't exist
- Handle permission errors gracefully
- Log file path for debugging

### File Cleanup
- Delete preset.ini when instance is deleted
- Update preset.ini when instance is updated
- Don't leave orphaned files

### models_preset Precedence
- **User sets models_preset manually**: Always use that path, even if preset_ini content exists
  - Don't overwrite user's explicit choice
  - Both the auto-generated preset.ini and custom path can coexist
  - llama-server will use the path from models_preset
- **User provides preset_ini but no models_preset**: Auto-set models_preset to `{instance_dir}/preset.ini`
- **User updates preset_ini after manually setting models_preset**: Keep models_preset unchanged
  - User's manual setting takes precedence
  - Update the file at the auto-generated location anyway (for potential future use)
- **User clears preset_ini**: Don't clear models_preset if it was manually set

### Instance Updates
- When updating an instance with new preset_ini content:
  - Overwrite the existing `{instance_dir}/preset.ini` file
  - If models_preset was auto-set, keep it pointing to the same location
  - If models_preset was manually set, don't change it
- When instance is restarted, changes take effect

## Migration

No migration needed - this is a new feature. Existing instances continue to work without preset.ini.

