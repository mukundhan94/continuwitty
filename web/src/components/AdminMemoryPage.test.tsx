import { render, screen, waitFor } from '@testing-library/react'
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
})
