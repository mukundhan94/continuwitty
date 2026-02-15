import { useState } from 'react'
import type { FormEvent } from 'react'

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
    <div className="login-shell">
      <div className="login-card">
        <p className="eyebrow">Engram Vault</p>
        <h1>Chat Continuity Console</h1>
        <p className="support">Sign in with your local test account to access sessions and engrams.</p>
        <form className="login-form" onSubmit={handleSubmit}>
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
        {error ? <p className="error-line">{error}</p> : null}
      </div>
    </div>
  )
}
