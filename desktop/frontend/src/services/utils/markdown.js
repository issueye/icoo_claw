import MarkdownIt from 'markdown-it'

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: false,
})

markdown.renderer.rules.table_open = () => (
  '<div class="markdown-table-scroll scrollbar-thin" tabindex="0"><table>'
)
markdown.renderer.rules.table_close = () => '</table></div>'

export function renderMarkdown(value) {
  const source = String(value || '')
  if (!source.trim()) {
    return ''
  }
  return markdown.render(source)
}
