import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from '@/App'
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
  },
  serverApi: {
    getHelp: vi.fn(),
    getVersion: vi.fn(),
    getDevices: vi.fn(),
  }
}))

// Mock health service to avoid real network calls
vi.mock('@/lib/healthService', () => ({
  healthService: {
    subscribe: vi.fn(() => () => {}),
    checkHealth: vi.fn(),
  },
  checkHealth: vi.fn(),
}))

function renderApp() {
  return render(
    <AuthProvider>
      <InstancesProvider>
        <App />
      </InstancesProvider>
    </AuthProvider>
  )
}

describe('App Component - Critical Business Logic Only', () => {
  const mockInstances: Instance[] = [
    { name: 'test-instance-1', status: 'stopped', options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: 'model1.gguf' } } },
    { name: 'test-instance-2', status: 'running', options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: 'model2.gguf' } } }
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(instancesApi.list).mockResolvedValue(mockInstances)
    window.sessionStorage.setItem('llamactl_management_key', 'test-api-key-123')
    global.fetch = vi.fn(() => Promise.resolve(new Response(null, { status: 200 })))
    
    // Mock window.matchMedia for dark mode functionality
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('End-to-End Instance Management', () => {
    it('creates new instance with correct API call and updates UI', async () => {
      const user = userEvent.setup()
      const newInstance: Instance = {
        name: 'new-test-instance',
        status: 'stopped',
        options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: 'new-model.gguf' } }
      }
      vi.mocked(instancesApi.create).mockResolvedValue(newInstance)

      renderApp()

      // Wait for app to load
      await waitFor(() => {
        expect(screen.getByText('test-instance-1')).toBeInTheDocument()
      })

      // Complete create flow: button → form → API call → UI update
      await user.click(screen.getByText('Create Instance'))
      
      const nameInput = screen.getByLabelText(/Instance Name/)
      await user.type(nameInput, 'new-test-instance')
      
      await user.click(screen.getByTestId('dialog-save-button'))

      // Verify correct API call
      await waitFor(() => {
        expect(instancesApi.create).toHaveBeenCalledWith('new-test-instance', {
          auto_restart: true, // Default value
          backend_type: BackendType.LLAMA_CPP
        })
      })

      // Verify UI updates with new instance
      await waitFor(() => {
        expect(screen.getByText('new-test-instance')).toBeInTheDocument()
      })
    })

    it('updates existing instance with correct API call', async () => {
      const user = userEvent.setup()
      const updatedInstance: Instance = {
        name: 'test-instance-1',
        status: 'stopped',
        options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: 'updated-model.gguf' } }
      }
      vi.mocked(instancesApi.update).mockResolvedValue(updatedInstance)

      renderApp()

      await waitFor(() => {
        expect(screen.getByText('test-instance-1')).toBeInTheDocument()
      })

      // Complete edit flow: edit button → form → API call
      const editButtons = screen.getAllByTitle('Edit instance')
      await user.click(editButtons[0])
      
      await user.click(screen.getByTestId('dialog-save-button'))

      // Verify correct API call with existing instance data
      await waitFor(() => {
        expect(instancesApi.update).toHaveBeenCalledWith('test-instance-1', {
          backend_type: BackendType.LLAMA_CPP,
          backend_options: { model: "model1.gguf" } // Pre-filled from existing instance
        })
      })
    })

    it('renders instances and provides working interface', async () => {
      renderApp()

      // Verify the app loads instances and renders them
      await waitFor(() => {
        expect(screen.getByText('test-instance-1')).toBeInTheDocument()
        expect(screen.getByText('test-instance-2')).toBeInTheDocument()
        expect(screen.getByText('Instances (2)')).toBeInTheDocument()
      })

      // Verify action buttons are present (testing integration, not specific actions)
      expect(screen.getAllByTitle('Start instance').length).toBeGreaterThan(0)
      expect(screen.getAllByTitle('Stop instance').length).toBeGreaterThan(0)
      expect(screen.getAllByTitle('Edit instance').length).toBe(2)
      expect(screen.getAllByTitle('Delete instance').length).toBeGreaterThan(0)
    })

    it('delete confirmation calls correct API', async () => {
      const user = userEvent.setup()
      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
      vi.mocked(instancesApi.delete).mockResolvedValue(undefined)

      renderApp()

      await waitFor(() => {
        expect(screen.getByText('test-instance-1')).toBeInTheDocument()
      })

      const deleteButtons = screen.getAllByTitle('Delete instance')
      await user.click(deleteButtons[0])

      // Verify confirmation and API call
      expect(confirmSpy).toHaveBeenCalledWith('Are you sure you want to delete instance "test-instance-1"?')
      await waitFor(() => {
        expect(instancesApi.delete).toHaveBeenCalledWith('test-instance-1')
      })

      confirmSpy.mockRestore()
    })
  })

  describe('Error Handling', () => {
    it('handles instance loading errors gracefully', async () => {
      vi.mocked(instancesApi.list).mockRejectedValue(new Error('Failed to load instances'))

      renderApp()

      // App should still render and show error
      await waitFor(() => {
        expect(screen.getByText('Error loading instances')).toBeInTheDocument()
      })
    })

    it('shows empty state when no instances exist', async () => {
      vi.mocked(instancesApi.list).mockResolvedValue([])

      renderApp()

      await waitFor(() => {
        expect(screen.getByText('No instances found')).toBeInTheDocument()
      })
    })
  })
})