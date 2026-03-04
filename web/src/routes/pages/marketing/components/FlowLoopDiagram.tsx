import styled, { keyframes } from 'styled-components'

import { AnimatedSvgDiagram } from './AnimatedSvgDiagram'

const fadeIn = keyframes`
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
`

const drawPath = keyframes`
  to { stroke-dashoffset: 0; }
`

const flowDot = keyframes`
  from { offset-distance: 0%; }
  to { offset-distance: 100%; }
`

const Shell = styled.div`
  width: 100%;
  max-width: 700px;
  margin: 2rem auto 0;

  svg {
    width: 100%;
    height: auto;
  }

  .node { opacity: 0; }
  .conn { stroke-dasharray: 200; stroke-dashoffset: 200; }
  .flow-dot { opacity: 0; }

  &.visible {
    .node { animation: ${fadeIn} 500ms ease-out both; }
    .node-1 { animation-delay: 100ms; }
    .node-2 { animation-delay: 300ms; }
    .node-3 { animation-delay: 500ms; }
    .node-4 { animation-delay: 700ms; }

    .conn { animation: ${drawPath} 800ms ease-out both; }
    .conn-1 { animation-delay: 900ms; }
    .conn-2 { animation-delay: 1100ms; }
    .conn-3 { animation-delay: 1300ms; }
    .conn-loop { animation-delay: 1500ms; stroke-dasharray: 600; stroke-dashoffset: 600; animation-duration: 1200ms; }

    .flow-dot {
      opacity: 0.8;
      animation: ${flowDot} 3s linear infinite;
    }
    .flow-dot-1 { animation-delay: 2s; offset-path: path('M160,90 C190,90 210,90 240,90'); }
    .flow-dot-2 { animation-delay: 2.3s; offset-path: path('M310,90 C340,90 360,90 390,90'); }
    .flow-dot-3 { animation-delay: 2.6s; offset-path: path('M460,90 C490,90 510,90 540,90'); }
    .flow-dot-loop { animation-delay: 3s; offset-path: path('M570,120 C590,200 10,200 30,120'); animation-duration: 4s; }
  }

  @media (prefers-reduced-motion: reduce) {
    .node { opacity: 1; }
    .conn { stroke-dashoffset: 0; }
    .flow-dot { opacity: 0; }
    &.visible .flow-dot { opacity: 0; }
  }
`

function NodeBox({ x, label, icon, idx }: { x: number; label: string; icon: string; idx: number }) {
  return (
    <g className={`node node-${idx}`} transform={`translate(${x}, 55)`}>
      <rect width="120" height="70" rx="14" fill="rgba(12,53,71,0.6)" stroke="rgba(94,234,212,0.25)" strokeWidth="1" />
      <rect x="36" y="8" width="48" height="28" rx="8" fill="rgba(94,234,212,0.08)" stroke="rgba(94,234,212,0.15)" strokeWidth="0.75" />
      <text x="60" y="27" textAnchor="middle" fontFamily="JetBrains Mono, monospace" fontSize="14" fill="#5eead4">{icon}</text>
      <text x="60" y="55" textAnchor="middle" fontFamily="Comfortaa, sans-serif" fontWeight="600" fontSize="10" fill="#e8fffb">{label}</text>
    </g>
  )
}

export function FlowLoopDiagram({ className }: { className?: string }) {
  return (
    <AnimatedSvgDiagram className={className}>
      <Shell>
        <svg viewBox="0 0 600 220" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <linearGradient id="flowGrad" x1="0" y1="0" x2="1" y2="0">
              <stop offset="0%" stopColor="#0d9488" />
              <stop offset="100%" stopColor="#5eead4" />
            </linearGradient>
          </defs>

          {/* Connecting paths */}
          <path className="conn conn-1" d="M150,90 C180,90 200,90 240,90" stroke="url(#flowGrad)" strokeWidth="2" fill="none" strokeLinecap="round" />
          <path className="conn conn-2" d="M360,90 C390,90 410,90 390,90 L390,90" stroke="url(#flowGrad)" strokeWidth="2" fill="none" strokeLinecap="round" />
          <path className="conn conn-2" d="M300,90 C330,90 350,90 390,90" stroke="url(#flowGrad)" strokeWidth="2" fill="none" strokeLinecap="round" />
          <path className="conn conn-3" d="M450,90 C480,90 500,90 540,90" stroke="url(#flowGrad)" strokeWidth="2" fill="none" strokeLinecap="round" />

          {/* Return loop path */}
          <path className="conn conn-loop" d="M570,120 C590,190 10,190 30,120" stroke="rgba(94,234,212,0.15)" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeDasharray="6 4" />

          {/* Arrow on loop */}
          <polygon className="node node-1" points="30,117 36,125 24,125" fill="rgba(94,234,212,0.3)" style={{ animationDelay: '1800ms' }} />

          {/* Nodes */}
          <NodeBox x={30} label="Session" icon="+" idx={1} />
          <NodeBox x={180} label="Context" icon="?" idx={2} />
          <NodeBox x={330} label="Memory" icon="~" idx={3} />
          <NodeBox x={480} label="Continue" icon=">" idx={4} />

          {/* Arrows between nodes */}
          <polygon className="node node-2" points="237,87 243,90 237,93" fill="#5eead4" style={{ animationDelay: '950ms' }} />
          <polygon className="node node-3" points="387,87 393,90 387,93" fill="#5eead4" style={{ animationDelay: '1150ms' }} />
          <polygon className="node node-4" points="537,87 543,90 537,93" fill="#5eead4" style={{ animationDelay: '1350ms' }} />

          {/* Flow dots */}
          <circle className="flow-dot flow-dot-1" r="3" fill="#5eead4" />
          <circle className="flow-dot flow-dot-2" r="3" fill="#5eead4" />
          <circle className="flow-dot flow-dot-3" r="3" fill="#5eead4" />
          <circle className="flow-dot flow-dot-loop" r="3" fill="rgba(94,234,212,0.5)" />
        </svg>
      </Shell>
    </AnimatedSvgDiagram>
  )
}
