import React from 'react'
import { BackendType, type CreateInstanceOptions } from '@/types/instance'
import CheckboxInput from '@/components/form/CheckboxInput'
import TextInput from '@/components/form/TextInput'
import EnvVarsInput from '@/components/form/EnvVarsInput'
import { useBackendSettings } from '@/hooks/useConfig'

interface ExecutionContextSectionProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
}

const ExecutionContextSection: React.FC<ExecutionContextSectionProps> = ({
  formData,
  onChange
}) => {
  const backendSettings = useBackendSettings(formData.backend_type)

  // Get placeholder for command override based on backend type and config
  const getCommandPlaceholder = () => {
    if (backendSettings?.command) {
      return backendSettings.command
    }

    // Fallback placeholders if config is not loaded
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
    <div className="space-y-4">
      <h3 className="text-md font-medium">Execution Context</h3>

      {/* Docker Mode Toggle - only for backends that support Docker */}
      {formData.backend_type !== BackendType.MLX_LM && (
        <CheckboxInput
          id="docker_enabled"
          label="Enable Docker"
          value={formData.docker_enabled}
          onChange={(value) => onChange('docker_enabled', value)}
          description="Run backend in Docker container"
        />
      )}

      {/* Command Override - only shown when Docker is disabled or backend is MLX */}
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

export default ExecutionContextSection
