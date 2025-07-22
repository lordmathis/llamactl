import React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { CreateInstanceOptions } from '@/types/instance'
import { getFieldType, basicFieldsConfig } from '@/lib/zodFormUtils'

interface ZodFormFieldProps {
  fieldKey: keyof CreateInstanceOptions
  value: any
  onChange: (key: keyof CreateInstanceOptions, value: any) => void
}

const ZodFormField: React.FC<ZodFormFieldProps> = ({ fieldKey, value, onChange }) => {
  // Get configuration for basic fields, or use field name for advanced fields
  const config = basicFieldsConfig[fieldKey as string] || { label: fieldKey }
  
  // Get type from Zod schema
  const fieldType = getFieldType(fieldKey)

  const handleChange = (newValue: any) => {
    onChange(fieldKey, newValue)
  }

  const renderField = () => {
    switch (fieldType) {
      case 'boolean':
        return (
          <div className="flex items-center space-x-2">
            <Checkbox
              id={fieldKey}
              checked={value || false}
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
              {config.required && <span className="text-red-500 ml-1">*</span>}
            </Label>
            <Input
              id={fieldKey}
              type="number"
              value={value || ''}
              onChange={(e) => {
                const numValue = e.target.value ? parseFloat(e.target.value) : undefined
                handleChange(numValue)
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
              {config.required && <span className="text-red-500 ml-1">*</span>}
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
              {config.required && <span className="text-red-500 ml-1">*</span>}
            </Label>
            <Input
              id={fieldKey}
              type="text"
              value={value || ''}
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

export default ZodFormField