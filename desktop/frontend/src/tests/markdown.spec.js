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

  it('renders markdown tables', () => {
    const html = renderMarkdown('| 名称 | 说明 |\n| --- | --- |\n| `claw.exe` | claw 可执行程序 |')
    expect(html).toContain('<table>')
    expect(html).toContain('<th>名称</th>')
    expect(html).toContain('<code>claw.exe</code>')
  })
})
