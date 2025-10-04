import { instancesApi } from '@/lib/api'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('API Error Handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('converts HTTP errors to meaningful messages', async () => {
    const mockResponse = {
      ok: false,
      status: 409,
      text: () => Promise.resolve('Instance already exists'),
      clone: function() { return this }
    }
    mockFetch.mockResolvedValue(mockResponse)

    await expect(instancesApi.create('existing', {}))
      .rejects
      .toThrow('HTTP 409: Instance already exists')
  })

  it('handles empty error responses gracefully', async () => {
    const mockResponse = {
      ok: false,
      status: 500,
      text: () => Promise.resolve(''),
      clone: function() { return this }
    }
    mockFetch.mockResolvedValue(mockResponse)

    await expect(instancesApi.list())
      .rejects
      .toThrow('HTTP 500')
  })

  it('handles 204 No Content responses', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 204
    })

    const result = await instancesApi.delete('test-instance')
    expect(result).toBeUndefined()
  })

  it('builds query parameters correctly for logs', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve('logs')
    })

    await instancesApi.getLogs('test-instance', 100)

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringMatching(
        /^https?:\/\/[^/]+\/api\/v1\/instances\/test-instance\/logs\?lines=100$/
      ),
      expect.any(Object)
    )
  })
})