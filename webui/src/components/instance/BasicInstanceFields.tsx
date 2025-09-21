import React from 'react'
import { BackendType, type CreateInstanceOptions } from '@/types/instance'
import { getBasicFields, basicFieldsConfig } from '@/lib/zodFormUtils'
import { getFieldType } from '@/schemas/instanceOptions'
import TextInput from '@/components/form/TextInput'
import NumberInput from '@/components/form/NumberInput'
import CheckboxInput from '@/components/form/CheckboxInput'
import SelectInput from '@/components/form/SelectInput'

interface BasicInstanceFieldsProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: any) => void
}

const BasicInstanceFields: React.FC<BasicInstanceFieldsProps> = ({
  formData,
  onChange
}) => {
  const basicFields = getBasicFields()

  const renderField = (fieldKey: keyof CreateInstanceOptions) => {
    const config = basicFieldsConfig[fieldKey as string] || { label: fieldKey }
    const fieldType = getFieldType(fieldKey)

    // Special handling for backend_type field
    if (fieldKey === 'backend_type') {
      return (
        <SelectInput
          key={fieldKey}
          id={fieldKey}
          label={config.label}
          value={formData[fieldKey] || BackendType.LLAMA_CPP}
          onChange={(value) => onChange(fieldKey, value)}
          options={[
            { value: BackendType.LLAMA_CPP, label: 'Llama Server' },
            { value: BackendType.MLX_LM, label: 'MLX LM' },
            { value: BackendType.VLLM, label: 'vLLM' }
          ]}
          description={config.description}
        />
      )
    }

    // Render based on field type
    switch (fieldType) {
      case 'boolean':
        return (
          <CheckboxInput
            key={fieldKey}
            id={fieldKey}
            label={config.label}
            value={formData[fieldKey] as boolean | undefined}
            onChange={(value) => onChange(fieldKey, value)}
            description={config.description}
          />
        )

      case 'number':
        return (
          <NumberInput
            key={fieldKey}
            id={fieldKey}
            label={config.label}
            value={formData[fieldKey] as number | undefined}
            onChange={(value) => onChange(fieldKey, value)}
            placeholder={config.placeholder}
            description={config.description}
          />
        )

      default:
        return (
          <TextInput
            key={fieldKey}
            id={fieldKey}
            label={config.label}
            value={formData[fieldKey] as string | number | undefined}
            onChange={(value) => onChange(fieldKey, value)}
            placeholder={config.placeholder}
            description={config.description}
          />
        )
    }
  }

  // Filter out auto restart fields and backend_options (handled separately)
  const fieldsToRender = basicFields.filter(
    fieldKey => !['auto_restart', 'max_restarts', 'restart_delay', 'backend_options'].includes(fieldKey as string)
  )

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-medium">Basic Configuration</h3>
      {fieldsToRender.map(renderField)}
    </div>
  )
}

export default BasicInstanceFields