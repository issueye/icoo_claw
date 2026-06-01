export const THEME_OPTIONS = [
  {
    label: '深色青蓝',
    value: 'dark',
    description: '当前默认深色主题',
    colors: ['#22D3EE', '#0EA5E9', '#C084FC', '#070A12', '#F9FAFB'],
  },
  {
    label: '浅色青蓝',
    value: 'light',
    description: '当前默认浅色主题',
    colors: ['#0B7EA4', '#0F4F6A', '#A33B75', '#F8FAFC', '#0F172A'],
  },
  {
    label: '暖橙活力',
    value: 'sunset',
    description: '#FFC107 / #FF9800 / #FF5722',
    colors: ['#FFC107', '#FF9800', '#FF5722', '#FFFFFF', '#212121'],
  },
  {
    label: '极简灰阶',
    value: 'minimal-gray',
    description: '#212529 / #495057 / #ADB5BD',
    colors: ['#212529', '#495057', '#ADB5BD', '#DEE2E6', '#F8F9FA'],
  },
  {
    label: '蓝紫 SaaS',
    value: 'saas-blue',
    description: '#6366F1 / #818CF8 / #E0E7FF',
    colors: ['#6366F1', '#818CF8', '#E0E7FF', '#F8FAFC', '#1E293B'],
  },
]

const themeValues = new Set(THEME_OPTIONS.map((theme) => theme.value))

export function normalizeTheme(value) {
  const theme = String(value || '').trim()
  return themeValues.has(theme) ? theme : 'dark'
}

export function applyTheme(value) {
  const theme = normalizeTheme(value)
  localStorage.setItem('qq-theme', theme)
  document.documentElement.setAttribute('data-theme', theme)
  return theme
}

export function getStoredTheme() {
  return normalizeTheme(localStorage.getItem('qq-theme') || 'dark')
}

export function nextTheme(value) {
  const current = normalizeTheme(value)
  const index = THEME_OPTIONS.findIndex((theme) => theme.value === current)
  return THEME_OPTIONS[(index + 1) % THEME_OPTIONS.length].value
}
