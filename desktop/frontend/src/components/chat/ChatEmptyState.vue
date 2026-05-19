<template>
  <div class="flex flex-1 items-center justify-center px-6">
    <div class="qq-panel-strong max-w-2xl rounded-[8px] px-6 py-6">
      <p class="text-xs uppercase tracking-[0.24em] text-[color:var(--qq-text-tertiary)]">Chat First</p>
      <h2 class="mt-4 text-3xl font-semibold text-slate-50">先把对话主链路做实</h2>
      <p class="mt-4 text-sm leading-7 text-[color:var(--qq-text-secondary)]">
        当前版本只围绕聊天展开。左侧会话列表来自网关，首条消息会先在本地生成标题，再创建会话并通过 WebSocket 开始流式对话。
      </p>
    </div>
  </div>
</template>
