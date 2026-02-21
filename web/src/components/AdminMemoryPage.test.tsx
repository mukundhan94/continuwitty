import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import { lightTheme } from '../styles/theme'
import { AdminMemoryPage } from './AdminMemoryPage'

const memoryAdminMocks = vi.hoisted(() => ({
  listAdminSessions: vi.fn(async () => []),
  listAdminEngrams: vi.fn(async () => []),
  listCollections: vi.fn(async () => []),
}))

vi.mock('../api/memoryAdmin', async () => {
  const actual = await vi.importActual<typeof import('../api/memoryAdmin')>('../api/memoryAdmin')
  return {
    ...actual,
    listAdminSessions: memoryAdminMocks.listAdminSessions,
    listAdminEngrams: memoryAdminMocks.listAdminEngrams,
    listCollections: memoryAdminMocks.listCollections,
  }
})

describe('AdminMemoryPage', () => {
  it('renders management sections and loads admin datasets', async () => {
    render(
      <ThemeProvider theme={lightTheme}>
        <AdminMemoryPage
          projectId="engram-vault"
          onProjectChange={vi.fn()}
          onNotice={vi.fn()}
        />
      </ThemeProvider>,
    )

    expect(screen.getByText('Session Management')).toBeInTheDocument()
    expect(screen.getByText('Engram Management')).toBeInTheDocument()
    expect(screen.getByText('Collections')).toBeInTheDocument()

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminSessions).toHaveBeenCalled()
      expect(memoryAdminMocks.listAdminEngrams).toHaveBeenCalled()
      expect(memoryAdminMocks.listCollections).toHaveBeenCalled()
    })
  })

  it('reloads datasets with include_deleted when toggled', async () => {
    const user = userEvent.setup()
    render(
      <ThemeProvider theme={lightTheme}>
        <AdminMemoryPage
          projectId="engram-vault"
          onProjectChange={vi.fn()}
          onNotice={vi.fn()}
        />
      </ThemeProvider>,
    )

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminSessions).toHaveBeenCalled()
    })

    await user.click(screen.getByLabelText(/include deleted/i))

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminSessions).toHaveBeenLastCalledWith(
        expect.objectContaining({ include_deleted: true }),
      )
      expect(memoryAdminMocks.listAdminEngrams).toHaveBeenLastCalledWith(
        expect.objectContaining({ include_deleted: true }),
      )
      expect(memoryAdminMocks.listCollections).toHaveBeenLastCalledWith(
        expect.objectContaining({ include_deleted: true }),
      )
    })
  })
})
