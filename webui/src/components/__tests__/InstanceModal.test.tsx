import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import InstanceDialog from '@/components/InstanceDialog'
import { BackendType, type Instance } from '@/types/instance'

// Mock the ConfigContext helper hooks
vi.mock('@/hooks/useConfig', () => ({
  useInstanceDefaults: () => ({
    autoRestart: true,
    maxRestarts: 3,
    restartDelay: 5,
    onDemandStart: false,
  }),
  useBackendSettings: () => ({
    command: '/usr/bin/llama-server',
    dockerEnabled: false,
    dockerImage: '',
  }),
}))

describe('InstanceModal - Form Logic and Validation', () => {
  const mockOnSave = vi.fn()
  const mockOnOpenChange = vi.fn()

beforeEach(() => {
  vi.clearAllMocks()
  window.sessionStorage.setItem('llamactl_management_key', 'test-api-key-123')
  global.fetch = vi.fn(() => Promise.resolve(new Response(null, { status: 200 })))
})

afterEach(() => {
  vi.restoreAllMocks()
})

  describe('Create Mode', () => {
    it('validates instance name is required', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Try to submit without name
      const saveButton = screen.getByTestId('dialog-save-button')
      expect(saveButton).toBeDisabled()

      // Add name, button should be enabled
      const nameInput = screen.getByLabelText(/Instance Name/)
      await user.type(nameInput, 'test-instance')
      
      await waitFor(() => {
        expect(saveButton).not.toBeDisabled()
      })
    })

    it('validates instance name format', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      const nameInput = screen.getByLabelText(/Instance Name/)
      
      // Test invalid characters
      await user.type(nameInput, 'test instance!')
      
      expect(screen.getByText(/can only contain letters, numbers, hyphens, and underscores/)).toBeInTheDocument()
      expect(screen.getByTestId('dialog-save-button')).toBeDisabled()

      // Clear and test valid name
      await user.clear(nameInput)
      await user.type(nameInput, 'test-instance-123')
      
      await waitFor(() => {
        expect(screen.queryByText(/can only contain letters, numbers, hyphens, and underscores/)).not.toBeInTheDocument()
        expect(screen.getByTestId('dialog-save-button')).not.toBeDisabled()
      })
    })

    it('submits form with correct data structure', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Fill required name
      await user.type(screen.getByLabelText(/Instance Name/), 'my-instance')

      // Submit form
      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('my-instance', {
        auto_restart: true, // Default value from config
        backend_type: BackendType.LLAMA_CPP,
        docker_enabled: false,
        max_restarts: 3,
        on_demand_start: false,
        restart_delay: 5
      })
    })

    it('form resets when dialog reopens', async () => {
      const { rerender } = render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Fill form
      const nameInput = screen.getByLabelText(/Instance Name/)
      await userEvent.setup().type(nameInput, 'temp-name')

      // Close dialog
      rerender(
        <InstanceDialog
          open={false}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Reopen dialog
      rerender(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Form should be reset
      const newNameInput = screen.getByLabelText(/Instance Name/)
      expect(newNameInput).toHaveValue('')
    })
  })

  describe('Edit Mode', () => {
    const mockInstance: Instance = {
      id: 1,
      name: 'existing-instance',
      status: 'stopped',
      options: {
        backend_type: BackendType.LLAMA_CPP,
        backend_options: { model: 'test-model.gguf', gpu_layers: 10 },
        auto_restart: false
      }
    }

    it('pre-fills form with existing instance data', () => {
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
          instance={mockInstance}
        />
      )

      // Name should be pre-filled and disabled
      const nameInput = screen.getByDisplayValue('existing-instance')
      expect(nameInput).toBeDisabled()

      // Other fields should be pre-filled (where visible)
      // Note: Not all fields are easily testable without more complex setup
      expect(screen.getByText('Edit Instance')).toBeInTheDocument()
    })

    it('submits update with existing data when no changes made', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
          instance={mockInstance}
        />
      )

      // Submit without changes
      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('existing-instance', {
        backend_type: BackendType.LLAMA_CPP,
        backend_options: { model: 'test-model.gguf', gpu_layers: 10 },
        auto_restart: false
      })
    })

    it('shows correct button text for running vs stopped instances', () => {
      const runningInstance: Instance = { ...mockInstance, status: 'running' }

      const { rerender } = render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
          instance={mockInstance} // stopped
        />
      )

      expect(screen.getByTestId('dialog-save-button')).toBeInTheDocument()

      rerender(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
          instance={runningInstance} // running
        />
      )

      expect(screen.getByText('Update & Restart Instance')).toBeInTheDocument()
    })
  })

  describe('Auto Restart Configuration', () => {
    it('shows restart options when auto restart is enabled', () => {
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Auto restart should be enabled by default
      const autoRestartCheckbox = screen.getByLabelText(/Auto Restart/)
      expect(autoRestartCheckbox).toBeChecked()

      // Restart options should be visible
      expect(screen.getByLabelText(/Max Restarts/)).toBeInTheDocument()
      expect(screen.getByLabelText(/Restart Delay/)).toBeInTheDocument()
    })

    it('hides restart options when auto restart is disabled', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Disable auto restart
      const autoRestartCheckbox = screen.getByLabelText(/Auto Restart/)
      await user.click(autoRestartCheckbox)

      // Restart options should be hidden
      expect(screen.queryByLabelText(/Max Restarts/)).not.toBeInTheDocument()
      expect(screen.queryByLabelText(/Restart Delay/)).not.toBeInTheDocument()
    })

    it('includes restart options in form submission when enabled', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Fill form
      await user.type(screen.getByLabelText(/Instance Name/), 'test-instance')

      // Clear default values and set new restart options
      const maxRestartsInput = screen.getByLabelText(/Max Restarts/)
      const restartDelayInput = screen.getByLabelText(/Restart Delay/)
      await user.clear(maxRestartsInput)
      await user.type(maxRestartsInput, '5')
      await user.clear(restartDelayInput)
      await user.type(restartDelayInput, '10')

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('test-instance', {
        auto_restart: true,
        backend_type: BackendType.LLAMA_CPP,
        docker_enabled: false,
        max_restarts: 5,
        on_demand_start: false,
        restart_delay: 10
      })
    })
  })


  describe('Form Data Handling', () => {
    it('cleans up undefined values before submission', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Fill only required field
      await user.type(screen.getByLabelText(/Instance Name/), 'clean-instance')

      await user.click(screen.getByTestId('dialog-save-button'))

      // Should include default values from config
      expect(mockOnSave).toHaveBeenCalledWith('clean-instance', {
        auto_restart: true,
        backend_type: BackendType.LLAMA_CPP,
        docker_enabled: false,
        max_restarts: 3,
        on_demand_start: false,
        restart_delay: 5
      })
    })

    it('handles numeric fields correctly', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await user.type(screen.getByLabelText(/Instance Name/), 'numeric-test')

      // Test GPU layers field (numeric)
      const gpuLayersInput = screen.getByLabelText(/GPU Layers/)
      await user.type(gpuLayersInput, '15')

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('numeric-test', {
        auto_restart: true,
        backend_type: BackendType.LLAMA_CPP,
        backend_options: { gpu_layers: 15 }, // Should be number, not string
        docker_enabled: false,
        max_restarts: 3,
        on_demand_start: false,
        restart_delay: 5
      })
    })
  })

  describe('Modal Controls', () => {
    it('calls onOpenChange when cancel button clicked', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await user.click(screen.getByTestId('dialog-cancel-button'))

      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })

    it('calls onOpenChange after successful save', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await user.type(screen.getByLabelText(/Instance Name/), 'test')
      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalled()
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })
  })
})