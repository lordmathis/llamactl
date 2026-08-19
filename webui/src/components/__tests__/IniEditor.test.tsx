import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import IniEditor from '@/components/instance/IniEditor'
import { getLlamaFieldSuggestions } from '@/lib/llamaFieldSuggestions'

const [first, second] = getLlamaFieldSuggestions('m')

const SELECTED = 'bg-accent text-accent-foreground'

function rowOf(name: string): HTMLElement {
  return screen.getByText(name).closest('div') as HTMLElement
}

function renderEditor() {
  const onChange = vi.fn()
  const Harness = () => {
    const [value, setValue] = useState('')
    return <IniEditor value={value} onChange={(v) => { onChange(v); setValue(v) }} />
  }
  render(<Harness />)
  return { onChange }
}

describe('IniEditor - suggestion popup keyboard navigation', () => {
  it('moves selection to the second suggestion on ArrowDown and applies it on Enter', async () => {
    const user = userEvent.setup()
    const { onChange } = renderEditor()

    const textarea = screen.getByRole('textbox')
    await user.type(textarea, 'm')

    expect(rowOf(first.name).className).toContain(SELECTED)

    await user.keyboard('{ArrowDown}')

    expect(rowOf(second.name).className).toContain(SELECTED)
    expect(rowOf(first.name).className).not.toContain(SELECTED)

    await user.keyboard('{Enter}')

    expect(onChange).toHaveBeenCalledWith(`${second.name} = `)
  })

  it('keeps the popup closed after Escape', async () => {
    const user = userEvent.setup()
    renderEditor()

    const textarea = screen.getByRole('textbox')
    await user.type(textarea, 'm')
    expect(screen.getByText(first.name)).toBeInTheDocument()

    await user.keyboard('{Escape}')

    expect(screen.queryByText(first.name)).not.toBeInTheDocument()
  })
})
