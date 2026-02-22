import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import { lightTheme } from '../styles/theme'
import { SmartIdDropdown } from './SmartIdDropdown'

function optionValues(testId: string): string[] {
  const list = screen.getByTestId(testId)
  const options = list.querySelectorAll('option')
  return [...options].map((item) => item.getAttribute('value') || '')
}

describe('SmartIdDropdown', () => {
  it('deduplicates and ranks options using search text', () => {
    render(
      <ThemeProvider theme={lightTheme}>
        <SmartIdDropdown
          value="eng"
          options={['engram-vault', 'engram-vault', 'alpha', 'my-engram']}
          searchText="eng"
          onChange={vi.fn()}
          inputTestId="smart-id-input"
          optionsTestId="smart-id-options"
          matchCountTestId="smart-id-matches"
        />
      </ThemeProvider>,
    )

    expect(optionValues('smart-id-options')).toEqual(['engram-vault', 'my-engram'])
    expect(screen.getByTestId('smart-id-matches')).toHaveTextContent('2 matches')
  })

  it('propagates input changes', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    function Harness() {
      const [value, setValue] = useState('')
      return (
        <SmartIdDropdown
          id="project-id"
          value={value}
          options={['engram-vault']}
          onChange={(nextValue) => {
            onChange(nextValue)
            setValue(nextValue)
          }}
          inputTestId="smart-id-input"
        />
      )
    }

    render(
      <ThemeProvider theme={lightTheme}>
        <Harness />
      </ThemeProvider>,
    )

    await user.type(screen.getByTestId('smart-id-input'), 'engram-vault')
    expect(onChange).toHaveBeenCalled()
    expect(onChange).toHaveBeenLastCalledWith('engram-vault')
  })
})
