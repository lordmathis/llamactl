import React, { useState } from 'react'
import { BackendType, type CreateInstanceOptions } from '@/types/instance'
import { Button } from '@/components/ui/button'
import { Terminal, ChevronDown, ChevronRight } from 'lucide-react'
import { getBasicBackendFields } from '@/lib/zodFormUtils'
import BackendFormField from '@/components/BackendFormField'
import SelectInput from '@/components/form/SelectInput'
import CheckboxInput from '@/components/form/CheckboxInput'
import TextInput from '@/components/form/TextInput'
import EnvVarsInput from '@/components/form/EnvVarsInput'
import { useBackendSettings } from '@/hooks/useConfig'

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
  const backendSettings = useBackendSettings(formData.backend_type)
  const basicBackendFields = getBasicBackendFields(formData.backend_type)

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
        <div
          className="flex items-center justify-between cursor-pointer"
          onClick={() => setShowExecutionContext(!showExecutionContext)}
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
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default BackendTab
