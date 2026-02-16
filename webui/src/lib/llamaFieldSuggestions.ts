import {
  getAllLlamaCppFieldKeys,
  getLlamaCppFieldType,
  getAllLlamaCppAltKeys,
  getLlamaCppAltKeyType
} from '@/schemas/instanceOptions'

export interface FieldSuggestion {
  name: string
  type: 'text' | 'number' | 'boolean' | 'array'
}

function snakeToKebab(snake: string): string {
  return snake.replace(/_/g, '-')
}

export function getLlamaFieldSuggestions(input: string): FieldSuggestion[] {
  const allFields = getAllLlamaCppFieldKeys()
  const allAltKeys = getAllLlamaCppAltKeys()
  const query = input.toLowerCase().trim()

  const suggestionsMap = new Map<string, FieldSuggestion>()

  allFields
    .filter(field => field !== 'extra_args')
    .forEach(field => {
      const kebabName = snakeToKebab(field)
      suggestionsMap.set(kebabName, {
        name: kebabName,
        type: getLlamaCppFieldType(field)
      })
    })

  allAltKeys.forEach(altKey => {
    if (!suggestionsMap.has(altKey)) {
      suggestionsMap.set(altKey, {
        name: altKey,
        type: getLlamaCppAltKeyType(altKey)
      })
    }
  })

  const suggestions = Array.from(suggestionsMap.values())

  if (!query) return suggestions.slice(0, 20)

  return suggestions.filter(s => s.name.includes(query))
}
