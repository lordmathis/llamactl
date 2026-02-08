import { instancesApi } from '@/lib/api'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock fetch globally
const mockFetch = vi.fn()

describe('API Error Handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Set the mock fetch in beforeEach to ensure it overrides the setup.ts mock
    global.fetch = mockFetch
  })

  it('converts HTTP errors to meaningful messages', async () => {
    const mockResponse = new Response('Instance already exists', {
      status: 409,
      statusText: 'Conflict'
    })
    mockFetch.mockResolvedValue(mockResponse)

    await expect(instancesApi.create('existing', {}))
      .rejects
      .toThrow('HTTP 409: Instance already exists')
  })

  it('handles empty error responses gracefully', async () => {
    const mockResponse = new Response('', {
      status: 500,
      statusText: 'Internal Server Error'
    })
    mockFetch.mockResolvedValue(mockResponse)

    await expect(instancesApi.list())
      .rejects
      .toThrow('HTTP 500')
  })

  it('handles 204 No Content responses', async () => {
    const mockResponse = new Response(null, {
      status: 204,
      statusText: 'No Content'
    })
    mockFetch.mockResolvedValue(mockResponse)

    const result = await instancesApi.delete('test-instance')
    expect(result).toBeUndefined()
  })

  it('builds query parameters correctly for logs', async () => {
    const mockResponse = new Response('logs', {
      status: 200,
      statusText: 'OK'
    })
    mockFetch.mockResolvedValue(mockResponse)

    await instancesApi.getLogs('test-instance', 100)

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringMatching(
        /^https?:\/\/[^/]+\/api\/v1\/instances\/test-instance\/logs\?lines=100$/
      ),
      expect.any(Object)
    )
  })
})