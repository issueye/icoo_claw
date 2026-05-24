import MarkdownIt from 'markdown-it'

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: false,
})

export function renderMarkdown(value) {
  const source = String(value || '')
  if (!source.trim()) {
    return ''
  }
  return markdown.render(source)
}
