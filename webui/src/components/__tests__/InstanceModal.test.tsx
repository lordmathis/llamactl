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

      // Navigate to the advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      // Submit without changes
      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('existing-instance', {
        backend_type: BackendType.LLAMA_CPP,
        backend_options: { model: 'test-model.gguf', gpu_layers: 10 },
        auto_restart: false
      })
    })

    it('shows correct button text for running vs stopped instances', async () => {
      const user = userEvent.setup()
      const runningInstance: Instance = { ...mockInstance, status: 'running' }

      const { rerender } = render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
          instance={mockInstance} // stopped
        />
      )

      // Navigate to advanced tab to see the save button
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      expect(screen.getByTestId('dialog-save-button')).toBeInTheDocument()
      expect(screen.getByText('Update Instance')).toBeInTheDocument()

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
    it('shows restart options when auto restart is enabled', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      // Wait for the General tab to render (it's the default tab)
      await waitFor(() => {
        expect(screen.getByLabelText(/Instance Name/)).toBeInTheDocument()
      })

      // The Auto Restart Configuration section starts collapsed, so expand it
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

      // Wait for the General tab to render
      await waitFor(() => {
        expect(screen.getByLabelText(/Instance Name/)).toBeInTheDocument()
      })

      // Expand the Auto Restart Configuration section
      const autoRestartHeading = screen.getByText('Auto Restart Configuration')
      await user.click(autoRestartHeading)

      // Wait for the section to expand
      await waitFor(() => {
        expect(screen.getByRole('checkbox', { name: /Auto Restart/i })).toBeInTheDocument()
      })

      // Disable auto restart
      const autoRestartCheckbox = screen.getByRole('checkbox', { name: /Auto Restart/i })
      await user.click(autoRestartCheckbox)

      // Restart options should be hidden
      await waitFor(() => {
        expect(screen.queryByLabelText(/Max Restarts/)).not.toBeInTheDocument()
        expect(screen.queryByLabelText(/Restart Delay/)).not.toBeInTheDocument()
      })
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

      // Fill form - we start on General tab
      await user.type(screen.getByLabelText(/Instance Name/), 'test-instance')

      // Expand the Auto Restart Configuration section
      const autoRestartHeading = screen.getByText('Auto Restart Configuration')
      await user.click(autoRestartHeading)

      // Wait for fields to appear
      await waitFor(() => {
        expect(screen.getByLabelText(/Max Restarts/)).toBeInTheDocument()
      })

      // Modify restart options (these are on the General tab)
      const maxRestartsInput = screen.getByLabelText(/Max Restarts/)
      const restartDelayInput = screen.getByLabelText(/Restart Delay/)

      // Select all and replace
      await user.tripleClick(maxRestartsInput)
      await user.keyboard('5')
      await user.tripleClick(restartDelayInput)
      await user.keyboard('10')

      // Navigate to the advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

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

      // Navigate to the advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

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

      // Navigate to backend tab to access GPU layers field
      const backendTab = screen.getByRole('tab', { name: /Backend/i })
      await user.click(backendTab)

      // Test GPU layers field (numeric)
      const gpuLayersInput = screen.getByLabelText(/GPU Layers/)
      await user.type(gpuLayersInput, '15')

      // Navigate to the advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

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

      // Navigate to advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalled()
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })
  })

  describe('Preset Configuration', () => {
    it('includes preset_ini in form submission when provided', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await user.type(screen.getByLabelText(/Instance Name/), 'preset-instance')

      // Navigate to Preset tab (only visible for llama.cpp backend)
      const presetTab = screen.getByRole('tab', { name: /Preset/i })
      await user.click(presetTab)

      // Type preset content
      const presetTextarea = screen.getByPlaceholderText(/Edit your preset.ini file here/)
      await user.type(presetTextarea, '[model1]\nmodel = /path/to/model.gguf\n')

      // Navigate to advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('preset-instance', expect.objectContaining({
        preset_ini: '[model1]\nmodel = /path/to/model.gguf\n'
      }))
    })

    it('does not include preset_ini when empty', async () => {
      const user = userEvent.setup()

      render(
        <InstanceDialog
          open={true}
          onOpenChange={mockOnOpenChange}
          onSave={mockOnSave}
        />
      )

      await user.type(screen.getByLabelText(/Instance Name/), 'no-preset-instance')

      // Navigate to advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalledWith('no-preset-instance', expect.not.objectContaining({
        preset_ini: expect.anything()
      }))
    })
  })
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

      // Navigate to the advanced tab where the save button is
      const advancedTab = screen.getByRole('tab', { name: /Advanced/i })
      await user.click(advancedTab)

      await user.click(screen.getByTestId('dialog-save-button'))

      expect(mockOnSave).toHaveBeenCalled()
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })
  })
})