import '@testing-library/jest-dom'
import { afterEach, vi } from 'vitest'

// Mock fetch globally since your app uses fetch
global.fetch = vi.fn()

// Clean up after each test
afterEach(() => {
  vi.clearAllMocks()
})