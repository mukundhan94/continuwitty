export function normalizeTooltipMarkdown(input: string): string {
  const normalized = input.replace(/\r\n/g, '\n').trim()
  if (!normalized) {
    return ''
  }

  const withSectionBreaks = normalized
    .replace(/\s+---\s+/g, '\n\n---\n\n')
    .replace(/([^\n])\s+(?=#{1,4}\s)/g, '$1\n\n')

  const splitLeadingInlineTableBlocks = withSectionBreaks
    .split('\n')
    .flatMap((line) => {
      if (!(line.includes('|') && /\|\s*-{2,}/.test(line))) {
        return [line]
      }

      const firstPipeIndex = line.indexOf('|')
      if (firstPipeIndex > 0 && line.slice(0, firstPipeIndex).trim()) {
        return [line.slice(0, firstPipeIndex).trimEnd(), line.slice(firstPipeIndex).trimStart()]
      }
      return [line]
    })
    .join('\n')

  return splitLeadingInlineTableBlocks
    .split('\n')
    .map((line) => {
      if (!/\|\s*-{2,}/.test(line)) {
        return line
      }
      return line.replace(/\|\s+\|(?=\s*(?:[-:]{2,}|\d{1,2}:\d{2}|[*A-Za-z0-9[]))/g, '|\n|')
    })
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
}
