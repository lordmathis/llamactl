import React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface NumberInputProps {
  id: string
  label: string
  value: number | undefined
  onChange: (value: number | undefined) => void
  placeholder?: string
  description?: string
  disabled?: boolean
  className?: string
}

const NumberInput: React.FC<NumberInputProps> = ({
  id,
  label,
  value,
  onChange,
  placeholder,
  description,
  disabled = false,
  className
}) => {
  const handleChange = (inputValue: string) => {
    if (inputValue === '') {
      onChange(undefined)
      return
    }

    const numValue = parseFloat(inputValue)
    if (!isNaN(numValue)) {
      onChange(numValue)
    }
  }

  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>
        {label}
      </Label>
      <Input
        id={id}
        type="number"
        step="any"
        value={value !== undefined ? value : ''}
        onChange={(e) => handleChange(e.target.value)}
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

export default NumberInput