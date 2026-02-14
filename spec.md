# Implementation Spec: preset.ini Editor for llama.cpp Router Mode

## Overview

Add support for creating and editing `preset.ini` files within the instance dialog. This enables users to configure multiple models for llama.cpp router mode through a simple text editor with autocomplete suggestions.

## Goals

1. Allow users to manually create preset.ini configurations for llama.cpp instances
2. Provide autocomplete suggestions for field names based on existing schema
3. Store preset.ini content per-instance (not globally)
4. Support import/export of preset.ini files

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
Stored as string in instance options
  ↓
Sent to backend on instance create/update
  ↓
Backend writes to {instance_dir}/preset.ini
  ↓
llama-server reads preset.ini on startup
```

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

**Tests:** Add schema validation tests to verify preset_ini field is optional and accepts strings.

#### Field Suggestions

**File:** `webui/src/lib/llamaFieldSuggestions.ts` (new)

```typescript
import { 
  getAllLlamaCppFieldKeys, 
  getLlamaCppFieldType
} from '@/schemas/backends/llamacpp'

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

**Tests:** `webui/src/lib/llamaFieldSuggestions.test.ts` (new)
- Returns top 20 suggestions when input is empty
- Converts snake_case to kebab-case correctly
- Filters suggestions by partial match
- Excludes extra_args field from suggestions
- Includes correct type information for each field

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

**Tests:** `webui/src/components/instance/IniEditor.test.tsx` (new)
- Shows suggestions when typing field name
- Hides suggestions in section headers `[...]`
- Hides suggestions in comments `;`
- Applies suggestion on Enter/Tab
- Closes suggestions on Escape
- Keyboard navigation with arrows works correctly

#### Preset Tab

**File:** `webui/src/components/instance/PresetTab.tsx` (new)

UI Structure:
```
┌─────────────────────────────────────────┐
│ Preset Tab                              │
├─────────────────────────────────────────┤
│ Title: "Model Presets (preset.ini)"    │
│ Description: "Configure multiple..."    │
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

**Tests:** Verify JSON serialization/deserialization of preset_ini field.

#### Write preset.ini on Instance Creation

**File:** `pkg/instance/instance.go`

Add helper function to write preset.ini:

```go
// writePresetIni writes the preset.ini file if provided in options
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

        // Write preset.ini if provided
        if err := writePresetIni(name, opts, globalInstanceSettings.LogsDir); err != nil {
            log.Printf("Warning: Failed to write preset.ini for instance %s: %v", name, err)
            // Don't fail instance creation, just log the warning
        }
    }

    return instance
}
```

**Tests:** `pkg/instance/instance_test.go`
- Instance creates without preset.ini (existing behavior unchanged)
- Instance creates with preset.ini and writes file to correct location
- Instance creation logs warning if preset.ini write fails but doesn't fail creation
- Preset.ini file has correct content after write
- Preset.ini is written to correct directory structure (`{instance_dir}/preset.ini`)
- Empty or whitespace-only preset.ini doesn't create a file

**Documentation:** Update user documentation to explain router mode configuration via preset.ini tab.


## Edge Cases

### Empty Preset
- Empty string or whitespace-only → don't write preset.ini file
- Null/undefined → no preset.ini (default behavior)

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

## Migration

No migration needed - this is a new feature. Existing instances continue to work without preset.ini.

