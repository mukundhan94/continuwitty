import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import { appTheme } from '../styles/theme'
import { SaveEngramModal } from './SaveEngramModal'

describe('SaveEngramModal', () => {
  it('prefills abstract from provided default abstract', async () => {
    const onSave = vi.fn(async () => {})
    const user = userEvent.setup()

    render(
      <ThemeProvider theme={appTheme}>
        <SaveEngramModal
          defaultTitle="Incident Snapshot"
          defaultAbstract="DB saturation caused cascading payment latency."
          saving={false}
          onClose={vi.fn()}
          onSave={onSave}
        />
      </ThemeProvider>,
    )

    const abstractInput = screen.getByLabelText(/abstract/i)
    expect(abstractInput).toHaveValue('DB saturation caused cascading payment latency.')

    await user.click(screen.getByRole('button', { name: /save engram/i }))
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        abstract: 'DB saturation caused cascading payment latency.',
      }),
    )
  })
})
