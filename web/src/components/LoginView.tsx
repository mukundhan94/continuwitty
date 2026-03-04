import { useState } from 'react'
import type { FormEvent } from 'react'
import styled, { keyframes } from 'styled-components'

import { MemoryStrandMark } from './MemoryStrandMark'
import { ErrorText, EyebrowText, LoginCard, LoginShell, SupportText } from '../styles/primitives'

interface LoginViewProps {
  isSubmitting: boolean
  error: string | null
  onSubmit: (username: string, password: string) => Promise<void>
}

const fadeIn = keyframes`
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`

const HeroHeader = styled.div`
  display: grid;
  gap: 0.42rem;
  animation: ${fadeIn} 520ms ease;
`

const AnimatedBar = styled.div`
  position: relative;
  height: 20px;
  margin-top: 0.1rem;
  overflow: hidden;

  &::before,
  &::after {
    content: '';
    position: absolute;
    top: 0;
    width: 220%;
    height: 100%;
    background-repeat: repeat-x;
    background-size: 156px 20px;
    opacity: 0.32;
  }

  &::before {
    left: 0;
    background-image: radial-gradient(78px 18px at 50% 124%, rgba(94, 234, 212, 0.14), transparent 72%);
    animation: login-wave 24s linear infinite;
  }

  &::after {
    left: -8%;
    background-image: radial-gradient(78px 18px at 50% 124%, rgba(6, 182, 212, 0.1), transparent 72%);
    animation: login-wave 34s linear infinite reverse;
  }

  @keyframes login-wave {
    from {
      transform: translateX(0);
    }
    to {
      transform: translateX(-50%);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    &::before,
    &::after {
      animation: none;
    }
  }
`

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
        <HeroHeader>
          <MemoryStrandMark size="md" />
          <EyebrowText>ContinuWitty</EyebrowText>
          <h1 className="font-display text-3xl font-semibold tracking-tight text-ink">Intelligence that flows</h1>
          <SupportText>
            Sign in to continue your memory workspace. Agents and teams can pick up exactly where
            they left off.
          </SupportText>
          <AnimatedBar aria-hidden="true" />
        </HeroHeader>

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
