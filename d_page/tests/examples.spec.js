import { readFile } from 'node:fs/promises'
import { fileURLToPath, URL } from 'node:url'
import { describe, expect, it } from 'vitest'
import { createComponentRegistry } from '../src/registry/createComponentRegistry.js'
import { defaultComponents } from '../src/registry/defaultComponents.js'
import { validateSchema } from '../src/runtime/validateSchema.js'

const exampleNames = [
  'chat-tool-result.json',
  'simple-form.json',
  'table-card.json',
  'live-input-preview.json',
]

describe('example schemas', () => {
  it.each(exampleNames)('%s is a valid card tree schema', async (fileName) => {
    const schema = await readExampleSchema(fileName)
    const registry = createComponentRegistry(defaultComponents)
    const result = validateSchema(schema, { componentRegistry: registry })

    expect(result.errors).toEqual([])
    expect(result.valid).toBe(true)
  })
})

async function readExampleSchema(fileName) {
  const path = fileURLToPath(new URL(`../src/schemas/examples/${fileName}`, import.meta.url))
  return JSON.parse(await readFile(path, 'utf8'))
}
