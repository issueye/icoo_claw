<template>
  <div class="flex flex-1 items-center justify-center px-6">
    <div class="relative max-w-lg w-full">
      <!-- 背景光晕 -->
      <div class="absolute inset-0 -z-10 blur-3xl opacity-20 pointer-events-none">
        <div class="absolute top-0 left-1/4 h-32 w-32 rounded-full bg-[var(--qq-accent)]" />
        <div class="absolute bottom-0 right-1/4 h-24 w-24 rounded-full bg-[var(--qq-accent-pink)]" />
      </div>

      <div class="qq-panel-strong rounded-[12px] px-7 py-7 text-center">
        <!-- 图标 -->
        <div class="mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded-[10px] border border-white/10 bg-[linear-gradient(135deg,rgba(0,242,254,0.18),rgba(0,198,255,0.08))] shadow-[0_0_24px_rgba(0,242,254,0.12)]">
          <svg class="h-6 w-6 text-[var(--qq-accent)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 01.865-.501 48.172 48.172 0 003.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z" />
          </svg>
        </div>

        <h2 class="text-base font-semibold text-slate-100">开始新对话</h2>
        <p class="mt-2.5 text-sm leading-6 text-[color:var(--qq-text-tertiary)]">
          选择一个 Agent，在下方输入框输入你的问题，<br/>即可开始流式对话。
        </p>

        <!-- 分隔线 -->
        <div class="my-5 border-t border-white/8" />

        <div class="grid grid-cols-3 gap-3 text-left">
          <div class="rounded-[8px] border border-white/6 bg-white/3 p-3">
            <p class="text-[10px] font-semibold uppercase tracking-[0.12em] text-[color:var(--qq-accent)] opacity-80">流式输出</p>
            <p class="mt-1.5 text-[11px] leading-5 text-[color:var(--qq-text-tertiary)]">实时 WebSocket 增量响应</p>
          </div>
          <div class="rounded-[8px] border border-white/6 bg-white/3 p-3">
            <p class="text-[10px] font-semibold uppercase tracking-[0.12em] text-[color:var(--qq-accent)] opacity-80">本地标题</p>
            <p class="mt-1.5 text-[11px] leading-5 text-[color:var(--qq-text-tertiary)]">首条消息自动生成标题</p>
          </div>
          <div class="rounded-[8px] border border-white/6 bg-white/3 p-3">
            <p class="text-[10px] font-semibold uppercase tracking-[0.12em] text-[color:var(--qq-accent)] opacity-80">网关同步</p>
            <p class="mt-1.5 text-[11px] leading-5 text-[color:var(--qq-text-tertiary)]">会话记录由网关统一管理</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
