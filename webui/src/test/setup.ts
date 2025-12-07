import '@testing-library/jest-dom'
import { afterEach, beforeEach } from 'vitest'

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

  setItem(key: string, value: string): void {
    this.store.set(key, value)
  }
}

// Replace global localStorage
global.localStorage = new LocalStorageMock()

// Clean up before each test
beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  localStorage.clear()
})