import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import BackendFormField from '@/components/BackendFormField'
import { type CreateInstanceOptions } from '@/schemas/instanceOptions'

describe('BackendFormField - models_preset Field States', () => {
  const mockOnChange = vi.fn()

  describe('models_preset field with no preset_ini content', () => {
    it('shows default state when models_preset is empty and no preset_ini', () => {
      render(
        <BackendFormField
          fieldKey="models_preset"
          value=""
          onChange={mockOnChange}
          formData={{}}
        />
      )

      expect(screen.getByLabelText('Models Preset Path')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('/path/to/preset.ini')).toBeInTheDocument()
      expect(screen.getByText(/Optional: Path to preset.ini for router mode/)).toBeInTheDocument()
      expect(screen.queryByText('Auto')).not.toBeInTheDocument()
      expect(screen.queryByText('Custom')).not.toBeInTheDocument()
    })

    it('shows Auto badge when models_preset is empty but preset_ini has content', () => {
      const formData: CreateInstanceOptions = {
        preset_ini: '[model1]\nmodel = /path/to/model.gguf\n'
      }

      render(
        <BackendFormField
          fieldKey="models_preset"
          value=""
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.getByLabelText('Models Preset Path')).toBeInTheDocument()
      expect(screen.getByText('Auto')).toBeInTheDocument()
      expect(screen.getByText('Will be auto-set to the preset.ini created in Preset Editor')).toBeInTheDocument()
    })
  })

  describe('models_preset field with custom path', () => {
    it('shows Custom badge when models_preset has user-provided value', () => {
      const formData: CreateInstanceOptions = {
        preset_ini: '[model1]\nmodel = /path/to/model.gguf\n'
      }

      render(
        <BackendFormField
          fieldKey="models_preset"
          value="/custom/path/to/preset.ini"
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.getByLabelText('Models Preset Path')).toBeInTheDocument()
      expect(screen.getByText('Custom')).toBeInTheDocument()
      expect(screen.getByDisplayValue('/custom/path/to/preset.ini')).toBeInTheDocument()
      expect(screen.getByText(/Optional: Path to preset.ini for router mode/)).toBeInTheDocument()
    })

    it('shows Custom badge when models_preset has value but no preset_ini', () => {
      const formData: CreateInstanceOptions = {}

      render(
        <BackendFormField
          fieldKey="models_preset"
          value="/custom/path/to/preset.ini"
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.getByLabelText('Models Preset Path')).toBeInTheDocument()
      expect(screen.getByText('Custom')).toBeInTheDocument()
      expect(screen.getByDisplayValue('/custom/path/to/preset.ini')).toBeInTheDocument()
    })
  })

  describe('models_preset field input behavior', () => {
    it('calls onChange when user types in models_preset field', async () => {
      const user = userEvent.setup()

      render(
        <BackendFormField
          fieldKey="models_preset"
          value=""
          onChange={mockOnChange}
          formData={{}}
        />
      )

      const input = screen.getByLabelText('Models Preset Path')
      await user.type(input, '/my/preset.ini')

      expect(mockOnChange).toHaveBeenCalledWith('models_preset', '/my/preset.ini')
    })

    it('updates displayed value when prop changes', () => {
      const { rerender } = render(
        <BackendFormField
          fieldKey="models_preset"
          value="/first/path.ini"
          onChange={mockOnChange}
          formData={{}}
        />
      )

      expect(screen.getByDisplayValue('/first/path.ini')).toBeInTheDocument()

      rerender(
        <BackendFormField
          fieldKey="models_preset"
          value="/second/path.ini"
          onChange={mockOnChange}
          formData={{}}
        />
      )

      expect(screen.getByDisplayValue('/second/path.ini')).toBeInTheDocument()
      expect(screen.queryByDisplayValue('/first/path.ini')).not.toBeInTheDocument()
    })
  })

  describe('models_preset badge visibility transitions', () => {
    it('transitions from Auto to Custom when user types custom path', async () => {
      const user = userEvent.setup()
      let formData: CreateInstanceOptions = {
        preset_ini: '[model1]\nmodel = /path/to/model.gguf\n'
      }

      const { rerender } = render(
        <BackendFormField
          fieldKey="models_preset"
          value=""
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.getByText('Auto')).toBeInTheDocument()
      expect(screen.queryByText('Custom')).not.toBeInTheDocument()

      rerender(
        <BackendFormField
          fieldKey="models_preset"
          value="/custom/path.ini"
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.queryByText('Auto')).not.toBeInTheDocument()
      expect(screen.getByText('Custom')).toBeInTheDocument()
    })

    it('transitions from Custom to no badge when user clears value', () => {
      const formData: CreateInstanceOptions = {}

      const { rerender } = render(
        <BackendFormField
          fieldKey="models_preset"
          value="/custom/path.ini"
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.getByText('Custom')).toBeInTheDocument()

      rerender(
        <BackendFormField
          fieldKey="models_preset"
          value=""
          onChange={mockOnChange}
          formData={formData}
        />
      )

      expect(screen.queryByText('Custom')).not.toBeInTheDocument()
      expect(screen.queryByText('Auto')).not.toBeInTheDocument()
    })
  })
})
