import React, { useState } from 'react'
import { BackendType, type CreateInstanceOptions } from '@/types/instance'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Terminal, ChevronDown, ChevronRight } from 'lucide-react'
import { getBasicBackendFields, getAdvancedBackendFields } from '@/lib/zodFormUtils'
import BackendFormField from '@/components/BackendFormField'
import SelectInput from '@/components/form/SelectInput'

interface BackendConfigurationCardProps {
  formData: CreateInstanceOptions
  onBackendFieldChange: (key: string, value: unknown) => void
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
  onParseCommand: () => void
}

const BackendConfigurationCard: React.FC<BackendConfigurationCardProps> = ({
  formData,
  onBackendFieldChange,
  onChange,
  onParseCommand
}) => {
  const [showAdvanced, setShowAdvanced] = useState(false)
  const basicBackendFields = getBasicBackendFields(formData.backend_type)
  const advancedBackendFields = getAdvancedBackendFields(formData.backend_type)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Backend Configuration</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Backend Type Selection */}
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

        {/* Parse Command Section */}
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

        {/* Basic Backend Options */}
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

        {/* Advanced Backend Options */}
        {advancedBackendFields.length > 0 && (
          <div className="space-y-4">
            <Button
              variant="ghost"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-2 p-0 h-auto font-medium"
            >
              {showAdvanced ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              Advanced Backend Options
              <span className="text-muted-foreground text-sm font-normal">
                ({advancedBackendFields.length} options)
              </span>
            </Button>

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

        {/* Extra Arguments - Always visible */}
        <div className="space-y-4">
          <BackendFormField
            key="extra_args"
            fieldKey="extra_args"
            value={(formData.backend_options as Record<string, unknown>)?.extra_args as Record<string, string> | undefined}
            onChange={onBackendFieldChange}
          />
        </div>
      </CardContent>
    </Card>
  )
}

export default BackendConfigurationCard