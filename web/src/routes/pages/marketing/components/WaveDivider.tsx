import styled, { keyframes } from 'styled-components'

const drawOn = keyframes`
  to { stroke-dashoffset: 0; }
`

const Shell = styled.div`
  width: 100%;
  height: 40px;
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;

  svg {
    width: 100%;
    height: 100%;
  }

  path {
    stroke-dasharray: 1200;
    stroke-dashoffset: 1200;
    animation: ${drawOn} 2s ease-out forwards;
  }

  @media (prefers-reduced-motion: reduce) {
    path {
      animation: none;
      stroke-dashoffset: 0;
    }
  }
`

export function WaveDivider({ className }: { className?: string }) {
  return (
    <Shell className={className}>
      <svg viewBox="0 0 1000 20" preserveAspectRatio="none">
        <path
          d="M0,10 Q50,2 100,10 T200,10 T300,10 T400,10 T500,10 T600,10 T700,10 T800,10 T900,10 T1000,10"
          stroke="rgba(94,234,212,0.12)"
          strokeWidth="1.5"
          fill="none"
          strokeLinecap="round"
        />
      </svg>
    </Shell>
  )
}
