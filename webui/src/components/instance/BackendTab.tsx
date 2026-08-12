import type React from 'react'
import { useState } from 'react'
import { BackendType, type CreateInstanceOptions } from '@/types/instance'
import { Button } from '@/components/ui/button'
import { Terminal, ChevronDown, ChevronRight, Info } from 'lucide-react'
import { getBasicBackendFields } from '@/lib/zodFormUtils'
import BackendFormField from '@/components/BackendFormField'
import SelectInput from '@/components/form/SelectInput'
import CheckboxInput from '@/components/form/CheckboxInput'
import TextInput from '@/components/form/TextInput'
import EnvVarsInput from '@/components/form/EnvVarsInput'
import { useBackendSettings } from '@/hooks/useConfig'
import PresetDialog from './PresetDialog'

interface BackendTabProps {
  formData: CreateInstanceOptions
  onBackendFieldChange: (key: string, value: unknown) => void
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
  onParseCommand: () => void
}

const BackendTab: React.FC<BackendTabProps> = ({
  formData,
  onBackendFieldChange,
  onChange,
  onParseCommand
}) => {
  const [showExecutionContext, setShowExecutionContext] = useState(false)
  const [showPresetDialog, setShowPresetDialog] = useState(false)
  const backendSettings = useBackendSettings(formData.backend_type)
  const basicBackendFields = getBasicBackendFields(formData.backend_type)

  // Show router-mode callout for llama.cpp when no explicit model is configured
  const llamaCppOptions = formData.backend_options as Record<string, unknown> | undefined
  const showRouterModeCallout =
    formData.backend_type === BackendType.LLAMA_CPP &&
    !llamaCppOptions?.model &&
    !llamaCppOptions?.hf_repo &&
    !formData.preset_ini?.trim()

  const getCommandPlaceholder = () => {
    if (backendSettings?.command) {
      return backendSettings.command
    }

    switch (formData.backend_type) {
      case BackendType.LLAMA_CPP:
        return "llama-server"
      case BackendType.VLLM:
        return "vllm"
      case BackendType.MLX_LM:
        return "mlx_lm.server"
      default:
        return ""
    }
  }

  return (
    <div className="space-y-6 py-4">
      {showRouterModeCallout && (
        <div className="flex gap-3 rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950">
          <Info className="h-5 w-5 shrink-0 text-blue-500 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium text-blue-800 dark:text-blue-200">No model specified — router mode will be used</p>
            <p className="text-blue-700 dark:text-blue-300">
              Without a model, llama-server starts in <strong>router mode</strong> and
              auto-discovers any model presets cached in your Hugging Face cache directory.
              This is how multi-model instances work, but it can be surprising if you intended
              to load a single model.
            </p>
            <p className="text-blue-700 dark:text-blue-300">
              To load a specific model, set one of:
            </p>
            <ul className="list-disc list-inside text-blue-700 dark:text-blue-300 space-y-0.5">
              <li><strong>Model Path</strong> — local <code>.gguf</code> file (e.g. <code>/models/gemma-3-1b.gguf</code>)</li>
              <li><strong>HF Repo</strong> + <strong>HF File</strong> — download from Hugging Face on first start</li>
              <li><strong>Preset tab</strong> — define a <code>preset.ini</code> to intentionally configure multiple models for router mode</li>
            </ul>
          </div>
        </div>
      )}

      <SelectInput
        id="backend_type"
        label="Backend Type"
        value={formData.backend_type || BackendType.LLAMA_CPP}
        onChange={(value) => onChange('backend_type', value)}
        options={[
          { value: BackendType.LLAMA_CPP, label: 'Llama Server' },
          { value: BackendType.MLX_LM, label: 'MLX LM' },
          { value: BackendType.VLLM, label: 'vLLM' }
        ]}
        description="Select the backend server type"
      />

      <div className="space-y-4">
        {/* biome-ignore lint/a11y/useSemanticElements: toggle contains an <h3>, so a <button> would be invalid HTML; keyboard support provided via role/tabIndex/onKeyDown */}
        <div
          role="button"
          tabIndex={0}
          className="flex items-center justify-between cursor-pointer"
          onClick={() => setShowExecutionContext(!showExecutionContext)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              setShowExecutionContext(!showExecutionContext)
            }
          }}
        >
          <div className="flex items-center gap-2">
            {showExecutionContext ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <h3 className="text-md font-medium">Execution Context</h3>
          </div>
          {!showExecutionContext && (
            <div className="text-sm text-muted-foreground">
              {formData.docker_enabled && <span className="mr-3">Docker enabled</span>}
              {formData.command_override && <span className="mr-3">Custom command</span>}
              {formData.environment && Object.keys(formData.environment).length > 0 && (
                <span>{Object.keys(formData.environment).length} env var{Object.keys(formData.environment).length > 1 ? 's' : ''}</span>
              )}
              {!formData.docker_enabled && !formData.command_override && (!formData.environment || Object.keys(formData.environment).length === 0) && (
                <span>Default</span>
              )}
            </div>
          )}
        </div>

        {showExecutionContext && (
          <div className="space-y-4 pl-6 border-l-2 border-muted">
            {formData.backend_type !== BackendType.MLX_LM && (
              <CheckboxInput
                id="docker_enabled"
                label="Enable Docker"
                value={formData.docker_enabled}
                onChange={(value) => onChange('docker_enabled', value)}
                description="Run backend in Docker container"
              />
            )}

            {(formData.backend_type === BackendType.MLX_LM || formData.docker_enabled !== true) && (
              <TextInput
                id="command_override"
                label="Command Override"
                value={formData.command_override || ''}
                onChange={(value) => onChange('command_override', value)}
                placeholder={getCommandPlaceholder()}
                description="Custom path to backend executable (leave empty to use config default)"
              />
            )}

            <EnvVarsInput
              id="environment"
              label="Environment Variables"
              value={formData.environment}
              onChange={(value) => onChange('environment', value)}
              description="Custom environment variables for the instance"
            />
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <Button
          variant="outline"
          onClick={onParseCommand}
          className="flex items-center gap-2 w-fit"
        >
          <Terminal className="h-4 w-4" />
          Parse Command
        </Button>
        <p className="text-sm text-muted-foreground">
          Import settings from your backend command
        </p>
      </div>

      {basicBackendFields.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-md font-medium">Basic Backend Options</h3>
          {basicBackendFields.map((fieldKey) => (
            <BackendFormField
              key={fieldKey}
              fieldKey={fieldKey}
              value={(formData.backend_options as Record<string, unknown>)?.[fieldKey] as string | number | boolean | string[] | undefined}
              onChange={onBackendFieldChange}
              formData={formData}
              onOpenPresetDialog={fieldKey === 'models_preset' ? () => setShowPresetDialog(true) : undefined}
            />
          ))}
        </div>
      )}

      {formData.backend_type === BackendType.LLAMA_CPP && (
        <PresetDialog
          open={showPresetDialog}
          onOpenChange={setShowPresetDialog}
          formData={formData}
          onChange={onChange}
        />
      )}
    </div>
  )
}

export default BackendTab
