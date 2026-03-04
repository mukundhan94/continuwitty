import styled, { keyframes } from 'styled-components'

const drift = keyframes`
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
`

const Shell = styled.div`
  width: 100%;
  height: 56px;
  pointer-events: none;
  position: relative;
  overflow: hidden;
  opacity: 0.5;

  &::before,
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    width: 200%;
    background-repeat: repeat-x;
    background-size: 220px 56px;
  }

  &::before {
    background-image: radial-gradient(110px 26px at 50% 124%, rgba(94, 234, 212, 0.12), transparent 74%);
    animation: ${drift} 24s linear infinite;
  }

  &::after {
    background-image: radial-gradient(110px 26px at 50% 124%, rgba(6, 182, 212, 0.08), transparent 74%);
    animation: ${drift} 34s linear infinite reverse;
  }

  @media (prefers-reduced-motion: reduce) {
    &::before,
    &::after {
      animation: none;
    }
  }
`

export function WaveDivider({ className }: { className?: string }) {
  return <Shell className={className} />
}
