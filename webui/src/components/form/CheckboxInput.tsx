import React from 'react'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'

interface CheckboxInputProps {
  id: string
  label: string
  value: boolean | undefined
  onChange: (value: boolean) => void
  description?: string
  disabled?: boolean
  className?: string
}

const CheckboxInput: React.FC<CheckboxInputProps> = ({
  id,
  label,
  value,
  onChange,
  description,
  disabled = false,
  className
}) => {
  return (
    <div className={`flex items-center space-x-2 ${className || ''}`}>
      <Checkbox
        id={id}
        checked={value === true}
        onCheckedChange={(checked) => onChange(!!checked)}
        disabled={disabled}
      />
      <Label htmlFor={id} className="text-sm font-normal">
        {label}
        {description && (
          <span className="text-muted-foreground ml-1">- {description}</span>
        )}
      </Label>
    </div>
  )
}

export default CheckboxInput