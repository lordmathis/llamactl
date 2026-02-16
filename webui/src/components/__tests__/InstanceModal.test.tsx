import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import InstanceDialog from '@/components/InstanceDialog'
import { BackendType, type Instance } from '@/types/instance'

// Mock the API module
vi.mock('@/lib/api', () => ({
  nodesApi: {
    list: vi.fn(() => Promise.resolve({})),
  },
}))

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

      // Navigate to the last tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      // Try to submit without name - save button should be disabled
      const saveButton = screen.getByTestId('dialog-save-button')
      expect(saveButton).toBeDisabled()

      // Go back to general tab to add name
      const generalTab = screen.getByRole('tab', { name: /General/i })
      await user.click(generalTab)

      // Add name
      const nameInput = screen.getByLabelText(/Instance Name/)
      await user.type(nameInput, 'test-instance')

      // Navigate back to advanced tab
      await user.click(advancedTab)

      await waitFor(() => {
        expect(screen.getByTestId('dialog-save-button')).not.toBeDisabled()
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

      // Navigate to advanced tab to check save button
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)
      expect(screen.getByTestId('dialog-save-button')).toBeDisabled()

      // Go back to general tab to fix the name
      const generalTab = screen.getByRole('tab', { name: /General/i })
      await user.click(generalTab)

      // Select all and delete, then type valid name
      const newNameInput = screen.getByLabelText(/Instance Name/)
      await user.tripleClick(newNameInput)
      await user.keyboard('{Backspace}')
      await user.type(newNameInput, 'test-instance-123')

      await waitFor(() => {
        expect(screen.queryByText(/can only contain letters, numbers, hyphens, and underscores/)).not.toBeInTheDocument()
      })

      // Navigate to advanced tab to check save button
      await user.click(advancedTab)
      await waitFor(() => {
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

      // Navigate to the advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      // Submit form
      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('my-instance', {
        auto_restart: true,
        backend_type: BackendType.LLAMA_CPP,
        docker_enabled: false,
        max_restarts: 3,
        on_demand_start: false,
        restart_delay: 5
      })
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
      expect(screen.getByText('Edit Instance')).toBeInTheDocument()
    })

    it('shows correct button text for running instances', async () => {
      const user = userEvent.setup()
      const runningInstance: Instance = { ...mockInstance, status: 'running' }

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
          instance={runningInstance}
        />
      )

      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      expect(screen.getByText('Update & Restart Instance')).toBeInTheDocument()
    })
  })

  describe('Auto Restart Configuration', () => {
    it('shows and hides restart options based on auto restart checkbox', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await waitFor(() => {
        expect(screen.getByLabelText(/Instance Name/)).toBeInTheDocument()
      })

      // Expand the Auto Restart Configuration section
      const autoRestartHeading = screen.getByText('Auto Restart Configuration')
      await user.click(autoRestartHeading)

      // Auto restart should be enabled by default
      await waitFor(() => {
        const autoRestartCheckbox = screen.getByRole('checkbox', { name: /Auto Restart/i })
        expect(autoRestartCheckbox).toBeChecked()
      })

      // Restart options should be visible
      expect(screen.getByLabelText(/Max Restarts/)).toBeInTheDocument()
      expect(screen.getByLabelText(/Restart Delay/)).toBeInTheDocument()

      // Disable auto restart
      const autoRestartCheckbox = screen.getByRole('checkbox', { name: /Auto Restart/i })
      await user.click(autoRestartCheckbox)

      // Restart options should be hidden
      await waitFor(() => {
        expect(screen.queryByLabelText(/Max Restarts/)).not.toBeInTheDocument()
        expect(screen.queryByLabelText(/Restart Delay/)).not.toBeInTheDocument()
      })
    })
  })

  describe('Modal Controls', () => {
    it('closes modal on cancel and after successful save', async () => {
      const user = userEvent.setup()

      const { rerender } = render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Test cancel
      await user.click(screen.getByTestId('dialog-cancel-button'))
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)

      // Reset and test save
      mockOnOpenChange.mockClear()
      rerender(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await user.type(screen.getByLabelText(/Instance Name/), 'test')

      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalled()
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })
  })
})
