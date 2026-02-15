import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { LoginView } from './LoginView'

describe('LoginView', () => {
  it('submits username and password values', async () => {
    const onSubmit = vi.fn(async () => {})
    const user = userEvent.setup()

    render(<LoginView isSubmitting={false} error={null} onSubmit={onSubmit} />)

    await user.clear(screen.getByLabelText(/username/i))
    await user.type(screen.getByLabelText(/username/i), 'analyst')
    await user.clear(screen.getByLabelText(/password/i))
    await user.type(screen.getByLabelText(/password/i), 'StrongPass123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(onSubmit).toHaveBeenCalledWith('analyst', 'StrongPass123')
  })
})
