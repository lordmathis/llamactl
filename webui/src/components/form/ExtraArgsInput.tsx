import React from 'react'
import KeyValueInput from './KeyValueInput'

interface ExtraArgsInputProps {
  id: string
  label: string
  value: Record<string, string> | undefined
  onChange: (value: Record<string, string> | undefined) => void
  description?: string
  disabled?: boolean
  className?: string
}

const ExtraArgsInput: React.FC<ExtraArgsInputProps> = (props) => {
  return (
    <KeyValueInput
      {...props}
      keyPlaceholder="Flag name (without --)"
      valuePlaceholder="Value (empty for boolean flags)"
      addButtonText="Add Argument"
      helperText="Additional command line arguments to pass to the backend. Leave value empty for boolean flags."
      allowEmptyValues={true}
    />
  )
}

export default ExtraArgsInput
