import React from 'react'
import KeyValueInput from './KeyValueInput'

interface EnvVarsInputProps {
  id: string
  label: string
  value: Record<string, string> | undefined
  onChange: (value: Record<string, string> | undefined) => void
  description?: string
  disabled?: boolean
  className?: string
}

const EnvVarsInput: React.FC<EnvVarsInputProps> = (props) => {
  return (
    <KeyValueInput
      {...props}
      keyPlaceholder="Variable name"
      valuePlaceholder="Variable value"
      addButtonText="Add Variable"
      allowEmptyValues={false}
    />
  )
}

export default EnvVarsInput
