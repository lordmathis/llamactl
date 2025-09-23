import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import InstanceCard from '@/components/InstanceCard'
import type { Instance } from '@/types/instance'
import { BackendType } from '@/types/instance'

// Mock the health hook since we're not testing health logic here
vi.mock('@/hooks/useInstanceHealth', () => ({
  useInstanceHealth: vi.fn(() => ({ status: 'ok', lastChecked: new Date() }))
}))

describe('InstanceCard - Instance Actions and State', () => {
  const mockStartInstance = vi.fn()
  const mockStopInstance = vi.fn()
  const mockDeleteInstance = vi.fn()
  const mockEditInstance = vi.fn()

  const stoppedInstance: Instance = {
    name: 'test-instance',
    status: 'stopped',
    options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: 'test-model.gguf' } }
  }

  const runningInstance: Instance = {
    name: 'running-instance',
    status: 'running',
    options: { backend_type: BackendType.LLAMA_CPP, backend_options: { model: 'running-model.gguf' } }
  }

beforeEach(() => {
  vi.clearAllMocks()
  window.sessionStorage.setItem('llamactl_management_key', 'test-api-key-123')
  global.fetch = vi.fn(() => Promise.resolve(new Response(null, { status: 200 })))
})

afterEach(() => {
  vi.restoreAllMocks()
})

  describe('Instance Action Buttons', () => {
    it('calls startInstance when start button clicked on stopped instance', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      const startButton = screen.getByTitle('Start instance')
      expect(startButton).not.toBeDisabled()
      
      await user.click(startButton)
      
      expect(mockStartInstance).toHaveBeenCalledWith('test-instance')
    })

    it('calls stopInstance when stop button clicked on running instance', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceCard
          instance={runningInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      const stopButton = screen.getByTitle('Stop instance')
      expect(stopButton).not.toBeDisabled()
      
      await user.click(stopButton)
      
      expect(mockStopInstance).toHaveBeenCalledWith('running-instance')
    })

    it('calls editInstance when edit button clicked', async () => {
      const user = userEvent.setup()
      
      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      const editButton = screen.getByTitle('Edit instance')
      await user.click(editButton)
      
      expect(mockEditInstance).toHaveBeenCalledWith(stoppedInstance)
    })

    it('opens logs dialog when logs button clicked', async () => {
      const user = userEvent.setup()

      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // First click "More actions" to reveal the logs button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      const logsButton = screen.getByTitle('View logs')
      await user.click(logsButton)

      // Should open logs dialog (we can verify this by checking if dialog title appears)
      expect(screen.getByText(`Logs: ${stoppedInstance.name}`)).toBeInTheDocument()
    })
  })

  describe('Delete Confirmation Logic', () => {
    it('shows confirmation dialog and calls deleteInstance when confirmed', async () => {
      const user = userEvent.setup()
      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // First click "More actions" to reveal the delete button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      const deleteButton = screen.getByTitle('Delete instance')
      await user.click(deleteButton)

      expect(confirmSpy).toHaveBeenCalledWith('Are you sure you want to delete instance "test-instance"?')
      expect(mockDeleteInstance).toHaveBeenCalledWith('test-instance')

      confirmSpy.mockRestore()
    })

    it('does not call deleteInstance when confirmation cancelled', async () => {
      const user = userEvent.setup()
      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)

      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // First click "More actions" to reveal the delete button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      const deleteButton = screen.getByTitle('Delete instance')
      await user.click(deleteButton)

      expect(confirmSpy).toHaveBeenCalled()
      expect(mockDeleteInstance).not.toHaveBeenCalled()

      confirmSpy.mockRestore()
    })
  })

  describe('Button State Based on Instance Status', () => {
    it('disables start button and enables stop button for running instance', async () => {
      const user = userEvent.setup()

      render(
        <InstanceCard
          instance={runningInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      expect(screen.queryByTitle('Start instance')).not.toBeInTheDocument()
      expect(screen.getByTitle('Stop instance')).not.toBeDisabled()

      // Expand more actions to access delete button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      expect(screen.getByTitle('Delete instance')).toBeDisabled() // Can't delete running instance
    })

    it('enables start button and disables stop button for stopped instance', async () => {
      const user = userEvent.setup()

      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      expect(screen.getByTitle('Start instance')).not.toBeDisabled()
      expect(screen.queryByTitle('Stop instance')).not.toBeInTheDocument()

      // Expand more actions to access delete button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      expect(screen.getByTitle('Delete instance')).not.toBeDisabled() // Can delete stopped instance
    })

    it('edit and logs buttons are always enabled', async () => {
      const user = userEvent.setup()

      render(
        <InstanceCard
          instance={runningInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      expect(screen.getByTitle('Edit instance')).not.toBeDisabled()

      // Expand more actions to access logs button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      expect(screen.getByTitle('View logs')).not.toBeDisabled()
    })
  })

  describe('Instance Information Display', () => {
    it('displays instance name correctly', () => {
      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      expect(screen.getByText('test-instance')).toBeInTheDocument()
    })

    it('shows health badge for running instances', () => {
      render(
        <InstanceCard
          instance={runningInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // Health badge should be present for running instances
      // The exact text depends on the health status from the mock
      expect(screen.getByText('Ready')).toBeInTheDocument()
    })

    it('does not show health badge for stopped instances', () => {
      render(
        <InstanceCard
          instance={stoppedInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // Health badge should not be present for stopped instances
      expect(screen.queryByText('Ready')).not.toBeInTheDocument()
    })
  })

  describe('Integration with LogsModal', () => {
    it('passes correct props to LogsModal', async () => {
      const user = userEvent.setup()

      render(
        <InstanceCard
          instance={runningInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // First click "More actions" to reveal the logs button
      const moreActionsButton = screen.getByTitle('More actions')
      await user.click(moreActionsButton)

      // Open logs dialog
      await user.click(screen.getByTitle('View logs'))

      // Verify dialog opened with correct instance data
      expect(screen.getByText('Logs: running-instance')).toBeInTheDocument()

      // Close dialog to test close functionality
      const closeButtons = screen.getAllByText('Close')
      const dialogCloseButton = closeButtons.find(button =>
        button.closest('[data-slot="dialog-content"]')
      )
      expect(dialogCloseButton).toBeTruthy()
      await user.click(dialogCloseButton!)

      // Modal should close
      expect(screen.queryByText('Logs: running-instance')).not.toBeInTheDocument()
    })
  })

  describe('Error Edge Cases', () => {
    it('handles instance with minimal data', () => {
      const minimalInstance: Instance = {
        name: 'minimal',
        status: 'stopped',
        options: {}
      }

      render(
        <InstanceCard
          instance={minimalInstance}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // Should still render basic structure
      expect(screen.getByText('minimal')).toBeInTheDocument()
      expect(screen.getByTitle('Start instance')).toBeInTheDocument()
    })

    it('handles instance with undefined options', () => {
      const instanceWithoutOptions: Instance = {
        name: 'no-options',
        status: 'running',
        options: undefined
      }

      render(
        <InstanceCard
          instance={instanceWithoutOptions}
          startInstance={mockStartInstance}
          stopInstance={mockStopInstance}
          deleteInstance={mockDeleteInstance}
          editInstance={mockEditInstance}
        />
      )

      // Should still work
      expect(screen.getByText('no-options')).toBeInTheDocument()
      expect(screen.getByTitle('Stop instance')).not.toBeDisabled()
    })
  })
})