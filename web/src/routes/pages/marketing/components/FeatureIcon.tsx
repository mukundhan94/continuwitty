const S = { stroke: '#5eead4', strokeWidth: 2, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const, fill: 'none' }

export function ChatIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  )
}

export function GraphIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <circle cx="6" cy="6" r="2.5" />
      <circle cx="18" cy="6" r="2.5" />
      <circle cx="12" cy="18" r="2.5" />
      <path d="M8.5 7l3 8.5M15.5 7l-3 8.5M8.5 6h7" />
    </svg>
  )
}

export function DocIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="8" y1="13" x2="16" y2="13" />
      <line x1="8" y1="17" x2="13" y2="17" />
    </svg>
  )
}

export function ShieldIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  )
}

export function KeyIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.78 7.78 5.5 5.5 0 0 1 7.78-7.78zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  )
}

export function CodeIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
    </svg>
  )
}

export function FlowIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M5 12h14" />
      <polyline points="12 5 19 12 12 19" />
      <circle cx="5" cy="12" r="1.5" fill="#5eead4" />
    </svg>
  )
}

export function ClockIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
  )
}

export function EyeIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  )
}

export function LockIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  )
}

export function PlugIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M12 2v6M8 2v4M16 2v4" />
      <rect x="6" y="8" width="12" height="6" rx="2" />
      <path d="M12 14v4M10 22h4" />
      <path d="M10 18h4" />
    </svg>
  )
}

export function LayersIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <polygon points="12 2 2 7 12 12 22 7 12 2" />
      <polyline points="2 17 12 22 22 17" />
      <polyline points="2 12 12 17 22 12" />
    </svg>
  )
}

export function CheckIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" stroke="#5eead4" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" fill="none">
      <circle cx="12" cy="12" r="10" strokeWidth="1.5" stroke="rgba(94,234,212,0.3)" />
      <polyline points="9 12 11.5 14.5 16 9.5" />
    </svg>
  )
}

export function ArrowLoopIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M17 1l4 4-4 4" />
      <path d="M3 11V9a4 4 0 0 1 4-4h14" />
      <path d="M7 23l-4-4 4-4" />
      <path d="M21 13v2a4 4 0 0 1-4 4H3" />
    </svg>
  )
}

export function SearchIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  )
}

export function UsersIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  )
}

export function BotIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" {...S}>
      <rect x="3" y="8" width="18" height="12" rx="3" />
      <circle cx="9" cy="14" r="1.5" fill="#5eead4" />
      <circle cx="15" cy="14" r="1.5" fill="#5eead4" />
      <path d="M12 2v4M8 6h8" />
    </svg>
  )
}
