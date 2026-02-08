import React, { useState } from 'react'
import { type CreateInstanceOptions } from '@/types/instance'
import { getAdvancedBackendFields } from '@/lib/zodFormUtils'
import BackendFormField from '@/components/BackendFormField'
import { ChevronDown, ChevronRight } from 'lucide-react'

interface AdvancedTabProps {
  formData: CreateInstanceOptions
  onBackendFieldChange: (key: string, value: unknown) => void
}

const AdvancedTab: React.FC<AdvancedTabProps> = ({
  formData,
  onBackendFieldChange
}) => {
  const [showAdvanced, setShowAdvanced] = useState(false)
  const advancedBackendFields = getAdvancedBackendFields(formData.backend_type)

  const extraArgs = (formData.backend_options as Record<string, unknown>)?.extra_args as Record<string, string> | undefined
  const advancedOptionsCount = advancedBackendFields.filter(fieldKey => {
    const value = (formData.backend_options as Record<string, unknown>)?.[fieldKey]
    return value !== undefined && value !== null && value !== ''
  }).length

  return (
    <div className="space-y-6 py-4">
      {advancedBackendFields.length > 0 && (
        <div className="space-y-4">
          <div
            className="flex items-center justify-between cursor-pointer"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            <h3 className="text-md font-medium">Advanced Backend Options</h3>
            <div className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
              {showAdvanced ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              {advancedOptionsCount} option{advancedOptionsCount !== 1 ? 's' : ''} set
            </div>
          </div>

          {showAdvanced && (
            <div className="space-y-4 pl-6 border-l-2 border-muted">
              {advancedBackendFields
                .sort()
                .map((fieldKey) => (
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
      )}

      <div className="space-y-4">
        <BackendFormField
          key="extra_args"
          fieldKey="extra_args"
          value={extraArgs}
          onChange={onBackendFieldChange}
        />
      </div>
    </div>
  )
}

export default AdvancedTab
