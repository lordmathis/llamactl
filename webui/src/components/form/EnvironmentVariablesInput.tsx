import React, { useState } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { X, Plus } from 'lucide-react'

interface EnvironmentVariablesInputProps {
  id: string
  label: string
  value: Record<string, string> | undefined
  onChange: (value: Record<string, string> | undefined) => void
  description?: string
  disabled?: boolean
  className?: string
}

interface EnvVar {
  key: string
  value: string
}

const EnvironmentVariablesInput: React.FC<EnvironmentVariablesInputProps> = ({
  id,
  label,
  value,
  onChange,
  description,
  disabled = false,
  className
}) => {
  // Convert the value object to an array of key-value pairs for editing
  const envVarsFromValue = value
    ? Object.entries(value).map(([key, val]) => ({ key, value: val }))
    : []

  const [envVars, setEnvVars] = useState<EnvVar[]>(
    envVarsFromValue.length > 0 ? envVarsFromValue : [{ key: '', value: '' }]
  )

  // Update parent component when env vars change
  const updateParent = (newEnvVars: EnvVar[]) => {
    // Filter out empty entries
    const validVars = newEnvVars.filter(env => env.key.trim() !== '' && env.value.trim() !== '')

    if (validVars.length === 0) {
      onChange(undefined)
    } else {
      const envObject = validVars.reduce((acc, env) => {
        acc[env.key.trim()] = env.value.trim()
        return acc
      }, {} as Record<string, string>)
      onChange(envObject)
    }
  }

  const handleKeyChange = (index: number, newKey: string) => {
    const newEnvVars = [...envVars]
    newEnvVars[index].key = newKey
    setEnvVars(newEnvVars)
    updateParent(newEnvVars)
  }

  const handleValueChange = (index: number, newValue: string) => {
    const newEnvVars = [...envVars]
    newEnvVars[index].value = newValue
    setEnvVars(newEnvVars)
    updateParent(newEnvVars)
  }

  const addEnvVar = () => {
    const newEnvVars = [...envVars, { key: '', value: '' }]
    setEnvVars(newEnvVars)
  }

  const removeEnvVar = (index: number) => {
    if (envVars.length === 1) {
      // Reset to empty if it's the last one
      const newEnvVars = [{ key: '', value: '' }]
      setEnvVars(newEnvVars)
      updateParent(newEnvVars)
    } else {
      const newEnvVars = envVars.filter((_, i) => i !== index)
      setEnvVars(newEnvVars)
      updateParent(newEnvVars)
    }
  }

  return (
    <div className={`grid gap-2 ${className || ''}`}>
      <Label htmlFor={id}>
        {label}
      </Label>
      <div className="space-y-2">
        {envVars.map((envVar, index) => (
          <div key={index} className="flex gap-2 items-center">
            <Input
              placeholder="Variable name"
              value={envVar.key}
              onChange={(e) => handleKeyChange(index, e.target.value)}
              disabled={disabled}
              className="flex-1"
            />
            <Input
              placeholder="Variable value"
              value={envVar.value}
              onChange={(e) => handleValueChange(index, e.target.value)}
              disabled={disabled}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => removeEnvVar(index)}
              disabled={disabled}
              className="shrink-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={addEnvVar}
          disabled={disabled}
          className="w-fit"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Variable
        </Button>
      </div>
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
      <p className="text-xs text-muted-foreground">
        Environment variables that will be passed to the backend process
      </p>
    </div>
  )
}

export default EnvironmentVariablesInput