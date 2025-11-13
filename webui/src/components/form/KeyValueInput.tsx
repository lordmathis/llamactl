import React, { useState, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { X, Plus } from 'lucide-react'

interface KeyValueInputProps {
  id: string
  label: string
  value: Record<string, string> | undefined
  onChange: (value: Record<string, string> | undefined) => void
  description?: string
  disabled?: boolean
  className?: string
  keyPlaceholder?: string
  valuePlaceholder?: string
  addButtonText?: string
  helperText?: string
  allowEmptyValues?: boolean // If true, entries with empty values are considered valid
}

interface KeyValuePair {
  key: string
  value: string
}

const KeyValueInput: React.FC<KeyValueInputProps> = ({
  id,
  label,
  value,
  onChange,
  description,
  disabled = false,
  className,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
  addButtonText = 'Add Entry',
  helperText,
  allowEmptyValues = false
}) => {
  // Convert the value object to an array of key-value pairs for editing
  const pairsFromValue = value
    ? Object.entries(value).map(([key, val]) => ({ key, value: val }))
    : []

  const [pairs, setPairs] = useState<KeyValuePair[]>(
    pairsFromValue.length > 0 ? pairsFromValue : [{ key: '', value: '' }]
  )

  // Sync internal state when value prop changes
  useEffect(() => {
    const newPairsFromValue = value
      ? Object.entries(value).map(([key, val]) => ({ key, value: val }))
      : []

    if (newPairsFromValue.length > 0) {
      setPairs(newPairsFromValue)
    } else if (!value) {
      // Reset to single empty row if value is explicitly undefined/null
      setPairs([{ key: '', value: '' }])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

  // Update parent component when pairs change
  const updateParent = (newPairs: KeyValuePair[]) => {
    // Filter based on validation rules
    const validPairs = allowEmptyValues
      ? newPairs.filter(pair => pair.key.trim() !== '')
      : newPairs.filter(pair => pair.key.trim() !== '' && pair.value.trim() !== '')

    if (validPairs.length === 0) {
      onChange(undefined)
    } else {
      const pairsObject = validPairs.reduce((acc, pair) => {
        acc[pair.key.trim()] = pair.value.trim()
        return acc
      }, {} as Record<string, string>)
      onChange(pairsObject)
    }
  }

  const handleKeyChange = (index: number, newKey: string) => {
    const newPairs = [...pairs]
    newPairs[index].key = newKey
    setPairs(newPairs)
    updateParent(newPairs)
  }

  const handleValueChange = (index: number, newValue: string) => {
    const newPairs = [...pairs]
    newPairs[index].value = newValue
    setPairs(newPairs)
    updateParent(newPairs)
  }

  const addPair = () => {
    const newPairs = [...pairs, { key: '', value: '' }]
    setPairs(newPairs)
  }

  const removePair = (index: number) => {
    if (pairs.length === 1) {
      // Reset to empty if it's the last one
      const newPairs = [{ key: '', value: '' }]
      setPairs(newPairs)
      updateParent(newPairs)
    } else {
      const newPairs = pairs.filter((_, i) => i !== index)
      setPairs(newPairs)
      updateParent(newPairs)
    }
  }

  return (
    <div className={`grid gap-2 ${className || ''}`}>
      <Label htmlFor={id}>
        {label}
      </Label>
      <div className="space-y-2">
        {pairs.map((pair, index) => (
          <div key={index} className="flex gap-2 items-center">
            <Input
              placeholder={keyPlaceholder}
              value={pair.key}
              onChange={(e) => handleKeyChange(index, e.target.value)}
              disabled={disabled}
              className="flex-1"
            />
            <Input
              placeholder={valuePlaceholder}
              value={pair.value}
              onChange={(e) => handleValueChange(index, e.target.value)}
              disabled={disabled}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => removePair(index)}
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
          onClick={addPair}
          disabled={disabled}
          className="w-fit"
        >
          <Plus className="h-4 w-4 mr-2" />
          {addButtonText}
        </Button>
      </div>
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
      {helperText && (
        <p className="text-xs text-muted-foreground">{helperText}</p>
      )}
    </div>
  )
}

export default KeyValueInput
