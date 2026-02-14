import {
  getAllLlamaCppFieldKeys,
  getLlamaCppFieldType
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
  const query = input.toLowerCase().trim()

  const suggestions: FieldSuggestion[] = allFields
    .filter(field => field !== 'extra_args')
    .map(field => ({
      name: snakeToKebab(field),
      type: getLlamaCppFieldType(field)
    }))

  if (!query) return suggestions.slice(0, 20)

  return suggestions.filter(s => s.name.includes(query))
}
