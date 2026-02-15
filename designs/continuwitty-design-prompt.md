# ContinuWitty — Design Language Prompt

> Copy this prompt and use it as a system instruction or context block when asking any LLM to generate UI for ContinuWitty.

---

## Brand Identity

- **Brand name:** ContinuWitty
- **Tagline:** "Intelligence that flows"
- **Core metaphor:** Ocean / flowing water / continuity as current
- **Personality:** Calm, trustworthy, intelligent, approachable, modern
- **Never:** Sharp, aggressive, loud, cluttered, corporate-cold

---

## Color System

```
Background (dark):     #0b1a2b  (deep ocean)
Surface:               #0c3547  (mid ocean)
Primary gradient:      linear-gradient(135deg, #0c4a6e, #0e7490, #0d9488, #14b8a6)
CTA / accent gradient: linear-gradient(135deg, #06b6d4, #14b8a6, #5eead4)
Seafoam (key accent):  #5eead4
Mint:                  #99f6e4
Text on dark:          rgba(178, 245, 234, 0.5) for body, #fff for headings
Background (light):    #f0fdfa or #fff
Text on light:         #134e4a for headings, #0c4a6e for body
Warm accent (sparse):  #fb923c (coral, for highlights only)
```

---

## Typography

| Role | Font | Weight | Notes |
|------|------|--------|-------|
| Headlines / Brand | Comfortaa | 600-700 | Rounded, geometric, friendly |
| Alt headlines | Quicksand | 500-700 | Lighter alternative |
| Body text | Nunito | 300-600 | Warm, highly readable |
| Code / technical | JetBrains Mono | 400-500 | Developer contexts |

**Rule:** All fonts must be rounded/soft. Never use sharp serif or condensed fonts. No Inter, Roboto, Arial, or system fonts.

---

## Shape Language

```
Border radii:  Always rounded — 8px (small), 14px (medium), 20px (large), full (pills/circles)
Corners:       NEVER sharp 0px corners. Minimum 8px.
Cards:         20px radius, 1px border rgba(94,234,212,0.08), backdrop-filter: blur(10px)
Buttons:       Full rounded (border-radius: 9999px) with gradient backgrounds
Icons:         Rounded stroke style (2-2.5px), stroke-linecap: round
```

---

## Wave & Motion System

Waves are the signature visual element. Use them as:
- Background decoration (SVG wave paths at bottom of sections)
- Underline accents (wavy line under headings)
- Dividers between sections
- Loading/ambient animations

**Wave SVG pattern:**
```html
<path d="M0,30 Q75,10 150,30 T300,30 T450,30 T600,30" />
```

**Animation timing:**
```
Waves:              8-15s linear infinite (slow, ambient, translateX)
Micro interactions: 200ms ease-out
Page transitions:   400ms ease-in-out
Ambient pulses:     4-6s ease-in-out infinite
Floating particles: 6-12s ease-in-out infinite
```

---

## Surface & Depth

**Glassmorphism (use sparingly):**
```css
background: rgba(11, 26, 43, 0.5);
backdrop-filter: blur(10px);
border: 1px solid rgba(94, 234, 212, 0.08);
```

**Ambient glow for hero sections:**
```css
radial-gradient(circle, rgba(14, 116, 144, 0.12) 0%, transparent 70%)
```

**Shadows are soft and teal-tinted:**
```css
box-shadow: 0 8px 24px rgba(13, 148, 136, 0.2);
```

---

## Logo Usage

- **Primary:** Icon (play symbol in frosted glass) + "ContinuWitty" in Comfortaa 700
- **"Witty" portion:** Always colored in `#5eead4` (dark bg) or `#0d9488` (light bg)
- **App icon:** Teal gradient with play symbol or water drop
- **Monogram:** "cW" in Comfortaa 700

**Recommended logo marks (in order of versatility):**
1. Water drop with play symbol inside — works as favicon, app icon, sticker
2. Ripple rings with center dot — animated or static, great for loading states
3. Frosted glass rounded square with play icon — tech-forward, modern
4. Wave underline below wordmark — simplest, works in any context
5. Bubble circle with play icon — consumer-friendly, approachable

---

## Component Patterns

```
Buttons:      Gradient bg (#06b6d4 → #14b8a6), white text, full rounded, subtle glow
Inputs:       Rounded, translucent bg, teal border on focus
Cards:        Glassmorphism, 20px radius, optional wave decoration at bottom
Navigation:   Clean, minimal, logo left, rounded pill-style nav items
Sections:     Generous padding (80px+ vertical), wave dividers between
Empty states: Animated ripple or floating particles
```

---

## Content Voice

- **Tone:** Calm, clear, slightly playful, confident without being pushy
- **Words to use:** flow, continue, pick up, stream, current, seamless
- **Words to avoid:** disrupt, break, hack, crush, dominate
- **CTAs:** "Start flowing", "Continue →", "Pick up where you left off"

---

## Don'ts

- ❌ Sharp corners or boxy layouts
- ❌ High-contrast neon or aggressive colors
- ❌ Dense, cluttered information hierarchy
- ❌ Generic sans-serif fonts (Inter, Roboto, Arial)
- ❌ Purple gradients (that's every other AI brand)
- ❌ Heavy drop shadows or hard borders
- ❌ Static, lifeless layouts — always add subtle motion
