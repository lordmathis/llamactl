import React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface TextInputProps {
  id: string
  label: string
  value: string | number | undefined
  onChange: (value: string | undefined) => void
  placeholder?: string
  description?: string
  disabled?: boolean
  className?: string
}

const TextInput: React.FC<TextInputProps> = ({
  id,
  label,
  value,
  onChange,
  placeholder,
  description,
  disabled = false,
  className
}) => {
  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>
        {label}
      </Label>
      <Input
        id={id}
        type="text"
        value={typeof value === 'string' || typeof value === 'number' ? value : ''}
        onChange={(e) => onChange(e.target.value || undefined)}
        placeholder={placeholder}
        disabled={disabled}
        className={className}
      />
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
    </div>
  )
}

export default TextInput