import React from 'react'
import { BackendType, type CreateInstanceOptions } from '@/types/instance'
import CheckboxInput from '@/components/form/CheckboxInput'
import TextInput from '@/components/form/TextInput'
import EnvVarsInput from '@/components/form/EnvVarsInput'
import SelectInput from '@/components/form/SelectInput'
import { useBackendSettings } from '@/hooks/useConfig'

interface ExecutionTabProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
}

const ExecutionTab: React.FC<ExecutionTabProps> = ({
  formData,
  onChange
}) => {
  const backendSettings = useBackendSettings(formData.backend_type)

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
  )
}

export default ExecutionTab
