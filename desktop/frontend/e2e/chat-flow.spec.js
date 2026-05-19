import { expect, test } from '@playwright/test'

test('chat flow creates, streams, and deletes a conversation', async ({ page }) => {
  const prompt = `playwright smoke ${Date.now()}`

  await page.addInitScript(() => {
    window.localStorage.clear()
  })

  await page.goto('/chat')

  await expect(page.getByText('网关已连接')).toBeVisible({ timeout: 20_000 })
  await expect(page.getByText('Agent Desktop Default Agent')).toBeVisible({ timeout: 20_000 })

  await page.getByTestId('chat-composer-input').fill(prompt)
  await page.getByTestId('chat-composer-send').click()

  await page.waitForURL(/\/chat\/conv_/)
  await expect(page.getByTestId('conversation-header-title')).toContainText(prompt)
  await expect(page.getByText(prompt, { exact: true })).toBeVisible({ timeout: 20_000 })
  await expect(page.getByText(`fake agent response: ${prompt}`, { exact: true })).toBeVisible({ timeout: 20_000 })

  const conversationId = page.url().split('/chat/')[1]
  await page.getByTestId(`conversation-delete-${conversationId}`).click()

  await page.waitForURL(/\/chat$/)
  await expect(page.getByText('先把对话主链路做实')).toBeVisible({ timeout: 10_000 })
})
