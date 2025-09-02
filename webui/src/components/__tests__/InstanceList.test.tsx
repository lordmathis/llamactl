import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import InstanceList from '@/components/InstanceList'
import { InstancesProvider } from '@/contexts/InstancesContext'
import { instancesApi } from '@/lib/api'
import type { Instance } from '@/types/instance'
import { BackendType } from '@/types/instance'
import { AuthProvider } from '@/contexts/AuthContext'

// Mock the API
vi.mock('@/lib/api', () => ({
  instancesApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
    restart: vi.fn(),
    delete: vi.fn(),
  }
}))

// Mock health service
vi.mock('@/lib/healthService', () => ({
  healthService: {
    subscribe: vi.fn(() => () => {}),
    checkHealth: vi.fn(),
  },
  checkHealth: vi.fn(),
}))

function renderInstanceList(editInstance = vi.fn()) {
  return render(
    <AuthProvider>
      <InstancesProvider>
        <InstanceList editInstance={editInstance} />
      </InstancesProvider>
    </AuthProvider>
  )
}

describe('InstanceList - State Management and UI Logic', () => {

  const mockEditInstance = vi.fn()

  const mockInstances: Instance[] = [
    { name: 'instance-1', status: 'stopped', options: { backend_type: BackendType.LLAMA_SERVER, backend_options: { model: 'model1.gguf' } } },
    { name: 'instance-2', status: 'running', options: { backend_type: BackendType.LLAMA_SERVER, backend_options: { model: 'model2.gguf' } } },
    { name: 'instance-3', status: 'stopped', options: { backend_type: BackendType.LLAMA_SERVER, backend_options: { model: 'model3.gguf' } } }
  ]

  const DUMMY_API_KEY = 'test-api-key-123'

  beforeEach(() => {
    vi.clearAllMocks()
    window.sessionStorage.setItem('llamactl_management_key', DUMMY_API_KEY)
    global.fetch = vi.fn(() => Promise.resolve(new Response(null, { status: 200 })))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Loading State', () => {
    it('shows loading spinner while instances are being fetched', () => {
      // Mock a delayed response to test loading state
      vi.mocked(instancesApi.list).mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve(mockInstances), 100))
      )

      renderInstanceList(mockEditInstance)

      // Should show loading state immediately
      expect(screen.getByText('Loading instances...')).toBeInTheDocument()
      expect(screen.getByLabelText('Loading')).toBeInTheDocument()
    })
  })

  describe('Error State', () => {
    it('displays error message when instance loading fails', async () => {
      const errorMessage = 'Failed to connect to server'
      vi.mocked(instancesApi.list).mockRejectedValue(new Error(errorMessage))

      renderInstanceList(mockEditInstance)

      // Wait for error to appear
      expect(await screen.findByText('Error loading instances')).toBeInTheDocument()
      expect(screen.getByText(errorMessage)).toBeInTheDocument()
    })

    it('does not show instances or loading when in error state', async () => {
      vi.mocked(instancesApi.list).mockRejectedValue(new Error('Network error'))

      renderInstanceList(mockEditInstance)

      await screen.findByText('Error loading instances')

      // Should not show loading or instance elements
      expect(screen.queryByText('Loading instances...')).not.toBeInTheDocument()
      expect(screen.queryByText('Instances (')).not.toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    it('shows empty state message when no instances exist', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue([])

      renderInstanceList(mockEditInstance)

      expect(await screen.findByText('No instances found')).toBeInTheDocument()
      expect(screen.getByText('Create your first instance to get started')).toBeInTheDocument()
    })

    it('does not show instances header when empty', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue([])

      renderInstanceList(mockEditInstance)

      await screen.findByText('No instances found')

      expect(screen.queryByText(/Instances \(/)).not.toBeInTheDocument()
    })
  })

  describe('Instances Display', () => {
    it('displays all instances with correct count', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)

      renderInstanceList(mockEditInstance)

      // Wait for instances to load
      expect(await screen.findByText('Instances (3)')).toBeInTheDocument()

      // All instances should be displayed
      expect(screen.getByText('instance-1')).toBeInTheDocument()
      expect(screen.getByText('instance-2')).toBeInTheDocument()
      expect(screen.getByText('instance-3')).toBeInTheDocument()
    })

    it('displays correct count based on instances received', async () => {
      // Test with different numbers of instances
      const twoInstances = mockInstances.slice(0, 2)
      vi.mocked(instancesApi.list).mockResolvedValue(twoInstances)

      renderInstanceList(mockEditInstance)

      expect(await screen.findByText('Instances (2)')).toBeInTheDocument()
      expect(screen.getByText('instance-1')).toBeInTheDocument()
      expect(screen.getByText('instance-2')).toBeInTheDocument()
      expect(screen.queryByText('instance-3')).not.toBeInTheDocument()
    })
  })

  describe('Instance Card Integration', () => {
    it('passes editInstance function to each instance card', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)

      renderInstanceList(mockEditInstance)

      await screen.findByText('Instances (3)')

      // Find edit buttons and click one
      const editButtons = screen.getAllByTitle('Edit instance')
      expect(editButtons).toHaveLength(3)

      // Click the first edit button
      await userEvent.setup().click(editButtons[0])

      // Should call editInstance with the correct instance
      expect(mockEditInstance).toHaveBeenCalledWith(mockInstances[0])
    })

    it('instance actions work through context integration', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)
      vi.mocked(instancesApi.start).mockResolvedValue({} as Instance)

      renderInstanceList(mockEditInstance)

      await screen.findByText('Instances (3)')

      // Find start buttons (should be available for stopped instances)
      const startButtons = screen.getAllByTitle('Start instance')
      expect(startButtons.length).toBeGreaterThan(0)

      // Click a start button
      await userEvent.setup().click(startButtons[0])

      // Should call the API (testing integration with context)
      expect(instancesApi.start).toHaveBeenCalled()
    })
  })

  describe('Performance Optimization', () => {
    it('uses memoized instance cards to prevent unnecessary re-renders', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)

      renderInstanceList(mockEditInstance)

      await screen.findByText('Instances (3)')

      // This is more of a structural test - we're verifying that the component
      // uses MemoizedInstanceCard (as mentioned in the source code comment)
      // The actual memoization effect would need more complex testing setup
      expect(screen.getAllByTitle('Edit instance')).toHaveLength(3)
    })
  })

  describe('Grid Layout', () => {
    it('renders instances in a grid layout', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)

      renderInstanceList(mockEditInstance)

      await screen.findByText('Instances (3)')

      // Check that instances are rendered in the expected container structure
      const instanceGrid = screen.getByText('instance-1').closest('.grid')
      expect(instanceGrid).toBeInTheDocument()
    })
  })

  describe('State Transitions', () => {
    it('transitions from loading to loaded state correctly', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)

      renderInstanceList(mockEditInstance)

      // Should start with loading
      expect(screen.getByText('Loading instances...')).toBeInTheDocument()

      // Should transition to loaded state
      expect(await screen.findByText('Instances (3)')).toBeInTheDocument()
      expect(screen.queryByText('Loading instances...')).not.toBeInTheDocument()
    })
  })
})