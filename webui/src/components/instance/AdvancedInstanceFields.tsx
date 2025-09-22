import React from 'react'
import type { CreateInstanceOptions } from '@/types/instance'
import { getAdvancedFields, basicFieldsConfig } from '@/lib/zodFormUtils'
import { getFieldType } from '@/schemas/instanceOptions'
import TextInput from '@/components/form/TextInput'
import NumberInput from '@/components/form/NumberInput'
import CheckboxInput from '@/components/form/CheckboxInput'
import ArrayInput from '@/components/form/ArrayInput'

interface AdvancedInstanceFieldsProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: any) => void
}

const AdvancedInstanceFields: React.FC<AdvancedInstanceFieldsProps> = ({
  formData,
  onChange
}) => {
  const advancedFields = getAdvancedFields()

  const renderField = (fieldKey: keyof CreateInstanceOptions) => {
    const config = basicFieldsConfig[fieldKey as string] || { label: fieldKey }
    const fieldType = getFieldType(fieldKey)

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

      case 'array':
        return (
          <ArrayInput
            key={fieldKey}
            id={fieldKey}
            label={config.label}
            value={formData[fieldKey] as string[] | undefined}
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

  // Filter out restart options and backend_options (handled separately)
  const fieldsToRender = advancedFields.filter(
    fieldKey => !['max_restarts', 'restart_delay', 'backend_options'].includes(fieldKey as string)
  )

  if (fieldsToRender.length === 0) {
    return null
  }

  return (
    <div className="space-y-4">
      <h4 className="text-md font-medium">Advanced Instance Configuration</h4>
      {fieldsToRender
        .sort()
        .map(renderField)}
    </div>
  )
}

export default AdvancedInstanceFields