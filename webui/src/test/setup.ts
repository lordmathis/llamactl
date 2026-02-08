import '@testing-library/jest-dom'
import { afterEach, beforeEach, vi } from 'vitest'

// Create a working localStorage implementation for tests
// This ensures localStorage works in both CLI and VSCode test runner
class LocalStorageMock implements Storage {
  private store: Map<string, string> = new Map()

  get length(): number {
    return this.store.size
  }

  clear(): void {
    this.store.clear()
  }

  getItem(key: string): string | null {
    return this.store.get(key) ?? null
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null
  }

  removeItem(key: string): void {
    this.store.delete(key)
  }

  setItem(key: string, value: string) {
    this.store.set(key, value)
  }
}

// Replace global localStorage
global.localStorage = new LocalStorageMock()

// Create a default fetch mock that handles common API endpoints
const createMockFetch = () => {
  return vi.fn((url: string) => {
    // Handle API endpoints that return JSON
    if (url.includes('/api/v1/')) {
      return Promise.resolve(
        new Response(JSON.stringify({}), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    }
    // Default response for other requests
    return Promise.resolve(
      new Response(null, { status: 200 })
    )
  })
}

// Clean up before each test
beforeEach(() => {
  localStorage.clear()
  // Set up default fetch mock
  global.fetch = createMockFetch() as typeof fetch
})

afterEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

afterEach(() => {
  localStorage.clear()
})