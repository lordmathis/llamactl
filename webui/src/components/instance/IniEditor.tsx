import React, { useState, useRef, useEffect, KeyboardEvent } from 'react'
import { Textarea } from '@/components/ui/textarea'
import { getLlamaFieldSuggestions, type FieldSuggestion } from '@/lib/llamaFieldSuggestions'

interface IniEditorProps {
  value: string
  onChange: (value: string) => void
  className?: string
}

const IniEditor: React.FC<IniEditorProps> = ({ value, onChange, className }) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [cursorPosition, setCursorPosition] = useState(0)
  const [suggestions, setSuggestions] = useState<FieldSuggestion[]>([])
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [suggestionPosition, setSuggestionPosition] = useState({ top: 0, left: 0 })

  const updateSuggestions = (text: string, cursorPos: number) => {
    const textBeforeCursor = text.substring(0, cursorPos)
    const currentLine = textBeforeCursor.split('\n').pop() || ''

    const trimmedLine = currentLine.trim()

    if (trimmedLine.startsWith('[') || trimmedLine.startsWith(';') || trimmedLine === '') {
      setShowSuggestions(false)
      return
    }

    const equalIndex = trimmedLine.indexOf('=')
    if (equalIndex !== -1) {
      setShowSuggestions(false)
      return
    }

    const fieldValue = trimmedLine
    if (fieldValue.length > 0) {
      const allSuggestions = getLlamaFieldSuggestions(fieldValue)
      setSuggestions(allSuggestions)
      setSelectedIndex(0)
      setShowSuggestions(allSuggestions.length > 0)

      if (textareaRef.current && allSuggestions.length > 0) {
        const rect = textareaRef.current.getBoundingClientRect()
        const lines = text.substring(0, cursorPos).split('\n')
        const currentLineNumber = lines.length
        const lineHeight = 21
        const top = rect.top + (currentLineNumber - 1) * lineHeight - textareaRef.current.scrollTop + 30
        const left = rect.left

        setSuggestionPosition({ top, left })
      }
    } else {
      setShowSuggestions(false)
    }
  }

  const applySuggestion = (suggestion: FieldSuggestion) => {
    const text = value
    const textBeforeCursor = text.substring(0, cursorPosition)
    const textAfterCursor = text.substring(cursorPosition)

    const lines = textBeforeCursor.split('\n')
    const lastLine = lines[lines.length - 1] || ''
    const linesBeforeLast = lines.slice(0, -1).join('\n')

    const leadingWhitespace = lastLine.match(/^\s*/)?.[0] || ''

    const newText = (linesBeforeLast ? linesBeforeLast + '\n' : '') +
                    leadingWhitespace +
                    suggestion.name +
                    ' = ' +
                    textAfterCursor

    onChange(newText)
    setShowSuggestions(false)

    setTimeout(() => {
      if (textareaRef.current) {
        const newCursorPos = (linesBeforeLast ? linesBeforeLast.length + 1 : 0) +
                            leadingWhitespace.length +
                            suggestion.name.length +
                            3
        textareaRef.current.focus()
        textareaRef.current.setSelectionRange(newCursorPos, newCursorPos)
      }
    }, 0)
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!showSuggestions) {
      return
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((prev) => (prev + 1) % suggestions.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((prev) => (prev - 1 + suggestions.length) % suggestions.length)
    } else if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      if (suggestions.length > 0) {
        applySuggestion(suggestions[selectedIndex])
      }
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setShowSuggestions(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value
    onChange(newValue)
    setCursorPosition(e.target.selectionStart)
    updateSuggestions(newValue, e.target.selectionStart)
  }

  const handleCursorChange = () => {
    if (textareaRef.current) {
      setCursorPosition(textareaRef.current.selectionStart)
      updateSuggestions(value, textareaRef.current.selectionStart)
    }
  }

  useEffect(() => {
    const textarea = textareaRef.current
    if (textarea) {
      textarea.addEventListener('keyup', handleCursorChange)
      textarea.addEventListener('click', handleCursorChange)
      return () => {
        textarea.removeEventListener('keyup', handleCursorChange)
        textarea.removeEventListener('click', handleCursorChange)
      }
    }
  }, [value])

  return (
    <div className={`relative ${className}`}>
      <Textarea
        ref={textareaRef}
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder="# Edit your preset.ini file here
# Field names will autocomplete as you type
# Example:
# model = /path/to/model.gguf
# gpu-layers = 35"
        className="font-mono text-sm min-h-[300px]"
        spellCheck={false}
      />
      {showSuggestions && suggestions.length > 0 && (
        <div
          className="fixed z-50 w-64 max-h-48 overflow-y-auto rounded-md border bg-popover p-1 shadow-md"
          style={{
            top: `${suggestionPosition.top}px`,
            left: `${suggestionPosition.left}px`
          }}
        >
          {suggestions.map((suggestion, index) => (
            <div
              key={suggestion.name}
              className={`flex flex-col rounded-sm px-2 py-1.5 text-sm cursor-pointer ${
                index === selectedIndex
                  ? 'bg-accent text-accent-foreground'
                  : 'hover:bg-accent/50'
              }`}
              onClick={() => applySuggestion(suggestion)}
            >
              <span className="font-medium">{suggestion.name}</span>
              <span className="text-xs text-muted-foreground capitalize">{suggestion.type}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default IniEditor
