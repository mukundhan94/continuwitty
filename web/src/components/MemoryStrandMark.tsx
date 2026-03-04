import { useId } from 'react'
import styled, { css, keyframes } from 'styled-components'

const draw = keyframes`
  to {
    stroke-dashoffset: 0;
  }
`

const fadeUp = keyframes`
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`

const pulse = keyframes`
  0% {
    opacity: 0.45;
  }
  50% {
    opacity: 0.95;
  }
  100% {
    opacity: 0.45;
  }
`

const MarkWrap = styled.div<{ $width: number; $height: number; $animated: boolean }>`
  width: ${({ $width }) => `${$width}px`};
  height: ${({ $height }) => `${$height}px`};
  opacity: 1;

  ${({ $animated }) =>
    $animated
      ? css`
          animation: ${fadeUp} 1.2s cubic-bezier(0.16, 1, 0.3, 1) 0.1s both;

          .strand1,
          .strand2,
          .strand-glow {
            stroke-dasharray: 400;
            stroke-dashoffset: 400;
          }

          .strand1 {
            animation: ${draw} 1.4s ease-out 0.2s forwards;
          }

          .strand2 {
            animation: ${draw} 1.4s ease-out 0.5s forwards;
          }

          .strand-glow {
            animation: ${draw} 1.8s ease-out 0.35s forwards;
          }

          .end-dot {
            animation: ${pulse} 2s ease-in-out infinite;
          }
        `
      : null}

  @media (prefers-reduced-motion: reduce) {
    animation: none;
    .strand1,
    .strand2,
    .strand-glow {
      animation: none;
      stroke-dasharray: 0;
      stroke-dashoffset: 0;
    }
    .end-dot {
      animation: none;
    }
  }
`

const MARK_SIZE = {
  sm: { width: 92, height: 33 },
  md: { width: 140, height: 50 },
  lg: { width: 196, height: 70 },
} as const

interface MemoryStrandMarkProps {
  size?: keyof typeof MARK_SIZE
  animated?: boolean
  className?: string
}

export function MemoryStrandMark({
  size = 'md',
  animated = true,
  className,
}: MemoryStrandMarkProps) {
  const dimensions = MARK_SIZE[size]
  const gradient1ID = useId()
  const gradient2ID = useId()
  const wispID = useId()
  const softGlowID = useId()
  return (
    <MarkWrap
      className={className}
      $width={dimensions.width}
      $height={dimensions.height}
      $animated={animated}
      aria-hidden="true"
    >
      <svg
        width={dimensions.width}
        height={dimensions.height}
        viewBox="0 0 140 50"
        style={{ filter: 'drop-shadow(0 0 24px rgba(94,234,212,0.3))' }}
      >
        <defs>
          <filter id={wispID} x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur in="SourceGraphic" stdDeviation="1.5" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
          <filter id={softGlowID} x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur in="SourceGraphic" stdDeviation="3" />
          </filter>
          <linearGradient id={gradient1ID} x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="#0d9488" />
            <stop offset="50%" stopColor="#14b8a6" />
            <stop offset="100%" stopColor="#5eead4" />
          </linearGradient>
          <linearGradient id={gradient2ID} x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="#5eead4" />
            <stop offset="50%" stopColor="#a7f3d0" />
            <stop offset="100%" stopColor="#ccfbf1" />
          </linearGradient>
        </defs>

        <path
          className="strand-glow"
          d="M6,28 C18,8 30,8 42,22 C54,36 62,38 76,22 C88,8 96,6 108,18 C118,28 126,32 136,20"
          stroke="rgba(94,234,212,0.08)"
          strokeWidth="12"
          fill="none"
          strokeLinecap="round"
          filter={`url(#${softGlowID})`}
        />

        <path
          className="strand1"
          d="M6,26 C18,6 30,6 42,20 C54,34 62,36 76,20 C88,6 96,4 108,16 C118,26 126,30 136,18"
          stroke={`url(#${gradient1ID})`}
          strokeWidth="3"
          fill="none"
          strokeLinecap="round"
          filter={`url(#${wispID})`}
        />

        <path
          className="strand2"
          d="M10,30 C22,12 33,11 44,24 C56,38 64,40 78,25 C90,11 98,10 110,21 C120,30 128,34 136,24"
          stroke={`url(#${gradient2ID})`}
          strokeWidth="2.2"
          fill="none"
          strokeLinecap="round"
          filter={`url(#${wispID})`}
        />

        <circle className="end-dot" cx="136" cy="18" r="3" fill="#5eead4" opacity="0.8" />
        <circle className="end-dot" cx="136" cy="24" r="2.5" fill="#ccfbf1" opacity="0.6" />
      </svg>
    </MarkWrap>
  )
}
