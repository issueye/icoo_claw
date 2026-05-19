# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: chat-flow.spec.js >> chat flow creates, streams, and deletes a conversation
- Location: e2e\chat-flow.spec.js:3:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('网关已连接')
Expected: visible
Timeout: 20000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 20000ms
  - waiting for getByText('网关已连接')

```

```yaml
- banner:
  - paragraph: Icoo Claw Desktop
  - heading "Gateway Chat Shell" [level=1]
  - text: 网关离线
  - button "刷新网关状态":
    - img
- complementary:
  - text: IC
  - navigation:
    - link "Chat":
      - /url: /chat
      - img
      - text: Chat
    - link "Search":
      - /url: /search
      - img
      - text: Search
    - link "Skills":
      - /url: /skills
      - img
      - text: Skills
    - link "Plugins":
      - /url: /plugins
      - img
      - text: Plugins
    - link "Automations":
      - /url: /automations
      - img
      - text: Automations
    - link "Settings":
      - /url: /settings
      - img
      - text: Settings
- complementary:
  - paragraph: Conversations
  - heading "会话列表" [level=2]
  - button "刷新":
    - img
  - button "新建会话":
    - img
  - text: 按最后活动时间排序 当前还没有会话。发送第一条消息后，会话会立即出现在这里。
- main:
  - text: Failed to fetch Gateway Offline Socket Idle Agent 未选择
  - paragraph: Chat First
  - heading "先把对话主链路做实" [level=2]
  - paragraph: 当前版本只围绕聊天展开。左侧会话列表来自网关，首条消息会先在本地生成标题，再创建会话并通过 WebSocket 开始流式对话。
  - textbox "输入你的问题，回车发送，Shift + Enter 换行"
  - paragraph: 聊天标题将使用首条用户输入在本地生成
  - button "发送" [disabled]:
    - img
    - text: 发送
```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test'
  2  | 
  3  | test('chat flow creates, streams, and deletes a conversation', async ({ page }) => {
  4  |   const prompt = `playwright smoke ${Date.now()}`
  5  | 
  6  |   await page.addInitScript(() => {
  7  |     window.localStorage.clear()
  8  |   })
  9  | 
  10 |   await page.goto('/chat')
  11 | 
> 12 |   await expect(page.getByText('网关已连接')).toBeVisible({ timeout: 20_000 })
     |                                         ^ Error: expect(locator).toBeVisible() failed
  13 |   await expect(page.getByText('Agent Desktop Default Agent')).toBeVisible({ timeout: 20_000 })
  14 | 
  15 |   await page.getByTestId('chat-composer-input').fill(prompt)
  16 |   await page.getByTestId('chat-composer-send').click()
  17 | 
  18 |   await page.waitForURL(/\/chat\/conv_/)
  19 |   await expect(page.getByTestId('conversation-header-title')).toContainText(prompt)
  20 |   await expect(page.getByText(prompt, { exact: true })).toBeVisible({ timeout: 20_000 })
  21 |   await expect(page.getByText(`fake agent response: ${prompt}`, { exact: true })).toBeVisible({ timeout: 20_000 })
  22 | 
  23 |   const conversationId = page.url().split('/chat/')[1]
  24 |   await page.getByTestId(`conversation-delete-${conversationId}`).click()
  25 | 
  26 |   await page.waitForURL(/\/chat$/)
  27 |   await expect(page.getByText('先把对话主链路做实')).toBeVisible({ timeout: 10_000 })
  28 | })
  29 | 
```