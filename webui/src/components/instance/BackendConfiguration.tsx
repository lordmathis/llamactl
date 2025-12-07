import React from 'react'
import type { CreateInstanceOptions } from '@/types/instance'
import { getBasicBackendFields, getAdvancedBackendFields } from '@/lib/zodFormUtils'
import BackendFormField from '@/components/BackendFormField'

type BackendFieldValue = string | number | boolean | string[] | Record<string, string> | undefined

interface BackendConfigurationProps {
  formData: CreateInstanceOptions
  onBackendFieldChange: (key: string, value: BackendFieldValue) => void
  showAdvanced?: boolean
}

const BackendConfiguration: React.FC<BackendConfigurationProps> = ({
  formData,
  onBackendFieldChange,
  showAdvanced = false
}) => {
  const basicBackendFields = getBasicBackendFields(formData.backend_type)
  const advancedBackendFields = getAdvancedBackendFields(formData.backend_type)

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-medium">Backend Configuration</h3>

      {/* Basic backend fields */}
      {basicBackendFields.map((fieldKey) => (
        <BackendFormField
          key={fieldKey}
          fieldKey={fieldKey}
          value={(formData.backend_options as Record<string, BackendFieldValue> | undefined)?.[fieldKey]}
          onChange={onBackendFieldChange}
        />
      ))}

      {/* Advanced backend fields */}
      {showAdvanced && advancedBackendFields.length > 0 && (
        <div className="space-y-4 pl-6 border-l-2 border-muted">
          <h4 className="text-md font-medium">Advanced Backend Configuration</h4>
          {advancedBackendFields
            .sort()
            .map((fieldKey) => (
              <BackendFormField
                key={fieldKey}
                fieldKey={fieldKey}
                value={(formData.backend_options as Record<string, BackendFieldValue> | undefined)?.[fieldKey]}
                onChange={onBackendFieldChange}
              />
            ))}
        </div>
      )}

      {/* Extra Args - Always visible as a separate section */}
      <div className="space-y-4">
        <BackendFormField
          key="extra_args"
          fieldKey="extra_args"
          value={(formData.backend_options as Record<string, BackendFieldValue> | undefined)?.extra_args}
          onChange={onBackendFieldChange}
        />
      </div>
    </div>
  )
}

export default BackendConfiguration
