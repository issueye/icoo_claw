import { describe, expect, it } from 'vitest'
import JSZip from 'jszip'
import { parseSkillMarkdown, parseSkillZip } from '@/services/utils/skill-package'

describe('skill package import', () => {
  it('parses SKILL.md frontmatter', () => {
    const parsed = parseSkillMarkdown(`---
name: doc-writer
description: Write docs
metadata:
  version: v2
---
Use this skill.`)

    expect(parsed.metadata.name).toBe('doc-writer')
    expect(parsed.metadata.description).toBe('Write docs')
    expect(parsed.metadata.metadata.version).toBe('v2')
    expect(parsed.body).toBe('Use this skill.')
  })

  it('extracts a zipped skill package with support files', async () => {
    const zip = new JSZip()
    zip.file(
      'doc-writer/SKILL.md',
      `---
name: doc-writer
description: Write docs
---
Use this skill.`,
    )
    zip.file('doc-writer/references/guide.md', 'guide')
    zip.file('doc-writer/assets/sample.txt', 'sample')
    zip.file('doc-writer/ignored.txt', 'ignored')

    const blob = await zip.generateAsync({ type: 'blob' })
    const file = new File([blob], 'doc-writer.zip', { type: 'application/zip' })
    const skill = await parseSkillZip(file)

    expect(skill.name).toBe('doc-writer')
    expect(skill.files.map((item) => item.path).sort()).toEqual(['assets/sample.txt', 'references/guide.md'])
  })

  it('normalizes display names from third-party skill packages', async () => {
    const zip = new JSZip()
    zip.file(
      'SKILL.md',
      `---
name: Agent Browser
description: Browser automation
metadata: {"version":"0.2.0","gateway_path":"tools/browser"}
---
Use this skill.`,
    )

    const blob = await zip.generateAsync({ type: 'blob' })
    const file = new File([blob], 'agent-browser-0.2.0.zip', { type: 'application/zip' })
    const skill = await parseSkillZip(file)

    expect(skill.name).toBe('agent-browser')
    expect(skill.path).toBe('tools/browser')
    expect(skill.version).toBe('0.2.0')
    expect(skill.metadata.original_name).toBe('Agent Browser')
  })
})
