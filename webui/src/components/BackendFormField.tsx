import React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { FileCode } from 'lucide-react'
import { getBackendFieldType, basicBackendFieldsConfig } from '@/lib/zodFormUtils'
import ExtraArgsInput from '@/components/form/ExtraArgsInput'
import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

interface BackendFormFieldProps {
  fieldKey: string
  value: string | number | boolean | string[] | Record<string, string> | undefined
  onChange: (key: string, value: string | number | boolean | string[] | Record<string, string> | undefined) => void
  formData?: CreateInstanceOptions
  onOpenPresetDialog?: () => void
}

const BackendFormField: React.FC<BackendFormFieldProps> = ({ fieldKey, value, onChange, formData, onOpenPresetDialog }) => {
  // Special handling for models_preset field
  if (fieldKey === 'models_preset') {
    const hasPresetContent = formData?.preset_ini && formData.preset_ini.trim().length > 0
    const isCustomPath = value && value.toString().trim().length > 0

    let helpText = 'Optional: Path to preset.ini for router mode'
    let badge = null

    if (!isCustomPath && hasPresetContent) {
      helpText = 'Will be auto-set to the preset.ini created in Preset Editor'
      badge = <Badge variant="secondary" className="ml-2">Auto</Badge>
    } else if (isCustomPath) {
      badge = <Badge variant="outline" className="ml-2">Custom</Badge>
    }

    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center">
            <Label htmlFor={fieldKey}>Models Preset Path</Label>
            {badge}
          </div>
          {onOpenPresetDialog && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onOpenPresetDialog}
              className="flex items-center gap-2"
            >
              <FileCode className="h-4 w-4" />
              Edit Preset
            </Button>
          )}
        </div>
        <Input
          id={fieldKey}
          value={value?.toString() || ''}
          onChange={(e) => onChange(fieldKey, e.target.value)}
          placeholder="/path/to/preset.ini"
        />
        <p className="text-sm text-muted-foreground">{helpText}</p>
      </div>
    )
  }

  // Special handling for extra_args
  if (fieldKey === 'extra_args') {
    return (
      <ExtraArgsInput
        id={fieldKey}
        label="Extra Arguments"
        value={value as Record<string, string> | undefined}
        onChange={(newValue) => onChange(fieldKey, newValue)}
        description="Additional command line arguments to pass to the backend"
      />
    )
  }

  // Get configuration for basic fields, or use field name for advanced fields
  const config = basicBackendFieldsConfig[fieldKey] || { label: fieldKey }

  // Get type from Zod schema
  const fieldType = getBackendFieldType(fieldKey)

  const handleChange = (newValue: string | number | boolean | string[] | undefined) => {
    onChange(fieldKey, newValue)
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