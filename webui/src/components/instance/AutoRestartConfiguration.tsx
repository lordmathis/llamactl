import React from 'react'
import type { CreateInstanceOptions } from '@/types/instance'
import CheckboxInput from '@/components/form/CheckboxInput'
import NumberInput from '@/components/form/NumberInput'

interface AutoRestartConfigurationProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: any) => void
}

const AutoRestartConfiguration: React.FC<AutoRestartConfigurationProps> = ({
  formData,
  onChange
}) => {
  const isAutoRestartEnabled = formData.auto_restart === true

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-medium">Auto Restart Configuration</h3>

      <CheckboxInput
        id="auto_restart"
        label="Auto Restart"
        value={formData.auto_restart}
        onChange={(value) => onChange('auto_restart', value)}
        description="Automatically restart the instance on failure"
      />

      {isAutoRestartEnabled && (
        <div className="ml-6 space-y-4 border-l-2 border-muted pl-4">
          <NumberInput
            id="max_restarts"
            label="Max Restarts"
            value={formData.max_restarts}
            onChange={(value) => onChange('max_restarts', value)}
            placeholder="3"
            description="Maximum number of restart attempts (0 = unlimited)"
          />
          <NumberInput
            id="restart_delay"
            label="Restart Delay (seconds)"
            value={formData.restart_delay}
            onChange={(value) => onChange('restart_delay', value)}
            placeholder="5"
            description="Delay in seconds before attempting restart"
          />
        </div>
      )}
    </div>
  )
}

export default AutoRestartConfiguration