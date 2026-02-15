import { useState } from 'react'
import type { FormEvent } from 'react'

import { ErrorText, EyebrowText, LoginCard, LoginShell, SupportText } from '../styles/primitives'

interface LoginViewProps {
  isSubmitting: boolean
  error: string | null
  onSubmit: (username: string, password: string) => Promise<void>
}

export function LoginView({ isSubmitting, error, onSubmit }: LoginViewProps) {
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('admin123')

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSubmit(username.trim(), password)
  }

  return (
    <LoginShell>
      <LoginCard className="space-y-4">
        <div className="space-y-1">
          <EyebrowText>Engram Vault</EyebrowText>
          <h1 className="font-display text-3xl font-semibold tracking-tight text-ink">Chat Continuity Console</h1>
          <SupportText>Sign in with your local test account to access sessions and engrams.</SupportText>
        </div>

        <form className="grid gap-2.5" onSubmit={handleSubmit}>
          <label htmlFor="username">Username</label>
          <input
            id="username"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
          />

          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />

          <button type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Signing in...' : 'Sign In'}
          </button>
        </form>

        {error ? <ErrorText className="mt-1">{error}</ErrorText> : null}
      </LoginCard>
    </LoginShell>
  )
}
