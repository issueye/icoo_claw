import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { compileStyle, parse } from '@vue/compiler-sfc'

const componentPath = fileURLToPath(new URL('../components/chat/ChatMessageItem.vue', import.meta.url))

describe('chat message markdown styles', () => {
  it('keeps light theme markdown colors scoped to rendered markdown descendants', () => {
    const source = readFileSync(componentPath, 'utf8')
    const { descriptor } = parse(source, { filename: componentPath })
    const style = descriptor.styles[0]
    const result = compileStyle({
      source: style.content,
      filename: componentPath,
      id: 'data-v-test',
      scoped: style.scoped,
    })

    expect(result.code).toContain('html[data-theme="light"] .markdown-body[data-v-test]')
    expect(result.code).toContain('color: var(--markdown-strong-color)')
    expect(result.code).toContain('color: var(--markdown-td-color)')
    expect(result.code).not.toMatch(/\[data-theme="light"\]\s+\.markdown-body\s*\{[^}]*border-color/)
  })
})
