import React, { useState } from 'react'
import type { CreateInstanceOptions } from '@/types/instance'
import CheckboxInput from '@/components/form/CheckboxInput'
import NumberInput from '@/components/form/NumberInput'
import { ChevronDown, ChevronRight } from 'lucide-react'

interface AutoRestartConfigurationProps {
  formData: CreateInstanceOptions
  onChange: <K extends keyof CreateInstanceOptions>(key: K, value: CreateInstanceOptions[K]) => void
}

const AutoRestartConfiguration: React.FC<AutoRestartConfigurationProps> = ({
  formData,
  onChange
}) => {
  const [isExpanded, setIsExpanded] = useState(false)
  const isAutoRestartEnabled = formData.auto_restart === true

  const getSummaryText = () => {
    if (!isAutoRestartEnabled) {
      return 'Disabled'
    }
    const maxRestarts = formData.max_restarts ?? 3
    const restartDelay = formData.restart_delay ?? 5
    const restartsText = maxRestarts === 0 ? 'unlimited' : `up to ${maxRestarts}`
    return `Enabled (restarts: ${restartsText}, delay: ${restartDelay}s)`
  }

  return (
    <div className="space-y-4">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <h3 className="text-lg font-medium">Auto Restart Configuration</h3>
        <div className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          {getSummaryText()}
        </div>
      </div>

      {isExpanded && (
        <div className="space-y-4 pl-6 border-l-2 border-muted">
          <CheckboxInput
            id="auto_restart"
            label="Auto Restart"
            value={formData.auto_restart}
            onChange={(value) => onChange('auto_restart', value)}
            description="Automatically restart the instance on failure"
          />

          {isAutoRestartEnabled && (
            <div className="space-y-4">
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
      )}
    </div>
  )
}

export default AutoRestartConfiguration
