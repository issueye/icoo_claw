import { describe, expect, it } from 'vitest'
import { renderMarkdown } from '@/services/utils/markdown'

describe('markdown rendering', () => {
  it('renders common markdown', () => {
    const html = renderMarkdown('## Title\n\n- **bold**')
    expect(html).toContain('<h2>Title</h2>')
    expect(html).toContain('<strong>bold</strong>')
  })

  it('escapes inline html', () => {
    const html = renderMarkdown('<script>alert(1)</script>')
    expect(html).not.toContain('<script>')
    expect(html).toContain('&lt;script&gt;')
  })
})
