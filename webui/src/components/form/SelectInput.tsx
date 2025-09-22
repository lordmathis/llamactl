import React from 'react'
import { Label } from '@/components/ui/label'

interface SelectOption {
  value: string
  label: string
}

interface SelectInputProps {
  id: string
  label: string
  value: string | undefined
  onChange: (value: string | undefined) => void
  options: SelectOption[]
  description?: string
  disabled?: boolean
  className?: string
}

const SelectInput: React.FC<SelectInputProps> = ({
  id,
  label,
  value,
  onChange,
  options,
  description,
  disabled = false,
  className
}) => {
  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>
        {label}
      </Label>
      <select
        id={id}
        value={value || ''}
        onChange={(e) => onChange(e.target.value || undefined)}
        disabled={disabled}
        className={`flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${className || ''}`}
      >
        {options.map(option => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
    </div>
  )
}

export default SelectInput