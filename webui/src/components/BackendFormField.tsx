import React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { getBackendFieldType, basicBackendFieldsConfig } from '@/lib/zodFormUtils'

interface BackendFormFieldProps {
  fieldKey: string
  value: string | number | boolean | string[] | undefined
  onChange: (key: string, value: string | number | boolean | string[] | undefined) => void
}

const BackendFormField: React.FC<BackendFormFieldProps> = ({ fieldKey, value, onChange }) => {
  // Get configuration for basic fields, or use field name for advanced fields
  const config = basicBackendFieldsConfig[fieldKey as string] || { label: fieldKey }
  
  // Get type from Zod schema
  const fieldType = getBackendFieldType(fieldKey)

  const handleChange = (newValue: string | number | boolean | string[] | undefined) => {
    onChange(fieldKey as string, newValue)
  }

  const renderField = () => {
    switch (fieldType) {
      case 'boolean':
        return (
          <div className="flex items-center space-x-2">
            <Checkbox
              id={fieldKey}
              checked={typeof value === 'boolean' ? value : false}
              onCheckedChange={(checked) => handleChange(checked)}
            />
            <Label htmlFor={fieldKey} className="text-sm font-normal">
              {config.label}
              {config.description && (
                <span className="text-muted-foreground ml-1">- {config.description}</span>
              )}
            </Label>
          </div>
        )

      case 'number':
        return (
          <div className="grid gap-2">
            <Label htmlFor={fieldKey}>
              {config.label}
            </Label>
            <Input
              id={fieldKey}
              type="number"
              step="any" // This allows decimal numbers
              value={typeof value === 'string' || typeof value === 'number' ? value : ''}
              onChange={(e) => {
                const numValue = e.target.value ? parseFloat(e.target.value) : undefined
                // Only update if the parsed value is valid or the input is empty
                if (e.target.value === '' || (numValue !== undefined && !isNaN(numValue))) {
                  handleChange(numValue)
                }
              }}
              placeholder={config.placeholder}
            />
            {config.description && (
              <p className="text-sm text-muted-foreground">{config.description}</p>
            )}
          </div>
        )

      case 'array':
        return (
          <div className="grid gap-2">
            <Label htmlFor={fieldKey}>
              {config.label}
            </Label>
            <Input
              id={fieldKey}
              type="text"
              value={Array.isArray(value) ? value.join(', ') : ''}
              onChange={(e) => {
                const arrayValue = e.target.value 
                  ? e.target.value.split(',').map(s => s.trim()).filter(Boolean)
                  : undefined
                handleChange(arrayValue)
              }}
              placeholder="item1, item2, item3"
            />
            {config.description && (
              <p className="text-sm text-muted-foreground">{config.description}</p>
            )}
            <p className="text-xs text-muted-foreground">Separate multiple values with commas</p>
          </div>
        )

      case 'text':
      default:
        return (
          <div className="grid gap-2">
            <Label htmlFor={fieldKey}>
              {config.label}
            </Label>
            <Input
              id={fieldKey}
              type="text"
              value={typeof value === 'string' || typeof value === 'number' ? value : ''}
              onChange={(e) => handleChange(e.target.value || undefined)}
              placeholder={config.placeholder}
            />
            {config.description && (
              <p className="text-sm text-muted-foreground">{config.description}</p>
            )}
          </div>
        )
    }
  }

  return <div className="space-y-2">{renderField()}</div>
}

export default BackendFormField