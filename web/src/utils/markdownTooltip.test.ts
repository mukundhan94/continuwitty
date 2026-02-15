import { describe, expect, it } from 'vitest'

import { normalizeTooltipMarkdown } from './markdownTooltip'

describe('normalizeTooltipMarkdown', () => {
  it('splits inline section boundaries and headings into markdown blocks', () => {
    const input = '# Primary Summary --- ## Next Section details'
    const normalized = normalizeTooltipMarkdown(input)

    expect(normalized).toContain('# Primary Summary')
    expect(normalized).toContain('---')
    expect(normalized).toContain('## Next Section details')
    expect(normalized).toContain('\n\n---\n\n')
  })

  it('keeps plain prose stable except line-ending normalization', () => {
    const input = 'Line one.\r\nLine two.'
    const normalized = normalizeTooltipMarkdown(input)
    expect(normalized).toBe('Line one.\nLine two.')
  })

  it('collapses excessive blank lines', () => {
    const input = '## Heading\n\n\n\nBody'
    expect(normalizeTooltipMarkdown(input)).toBe('## Heading\n\nBody')
  })

  it('converts inline table fragments into proper multi-line markdown tables', () => {
    const input =
      'Timeline | Time | Event | Owner | |------|-------|-------| | 14:32 | Alert fired | On-call | | 14:40 | Retry | SRE |'
    const normalized = normalizeTooltipMarkdown(input)

    expect(normalized).toContain('Timeline\n| Time | Event | Owner |')
    expect(normalized).toContain('\n|------|-------|-------|')
    expect(normalized).toContain('\n| 14:32 | Alert fired | On-call |')
    expect(normalized).toContain('\n| 14:40 | Retry | SRE |')
  })
})
