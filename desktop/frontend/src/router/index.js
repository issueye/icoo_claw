import { createRouter, createWebHistory } from 'vue-router'
import AppShell from '@/layouts/AppShell.vue'
import AgentsView from '@/views/AgentsView.vue'
import ChatConversationView from '@/views/ChatConversationView.vue'
import ChatHomeView from '@/views/ChatHomeView.vue'
import PlaceholderView from '@/views/PlaceholderView.vue'
import ProvidersView from '@/views/ProvidersView.vue'
import QqUedView from '@/views/QqUedView.vue'
import ScheduledTasksView from '@/views/ScheduledTasksView.vue'
import SearchView from '@/views/SearchView.vue'
import SettingsView from '@/views/SettingsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AppShell,
      children: [
        { path: '', redirect: '/chat' },
        { path: '/chat', name: 'chat-home', component: ChatHomeView },
        { path: '/chat/:id', name: 'chat-conversation', component: ChatConversationView },
        {
          path: '/ued',
          name: 'ued',
          component: QqUedView,
        },
        {
          path: '/search',
          name: 'search',
          component: SearchView,
        },
        {
          path: '/providers',
          name: 'providers',
          component: ProvidersView,
        },
        {
          path: '/agents',
          name: 'agents',
          component: AgentsView,
        },
        {
          path: '/skills',
          name: 'skills',
          component: PlaceholderView,
          meta: {
            eyebrow: '占位模块',
            title: 'Skills',
            description: '技能管理后续再接入，当前版本只保留导航和信息架子。',
          },
        },
        {
          path: '/plugins',
          name: 'plugins',
          component: PlaceholderView,
          meta: {
            eyebrow: '占位模块',
            title: 'Plugins',
            description: '插件页先作为结构占位，避免聊天主链路被旁支功能拖住。',
          },
        },
        {
          path: '/scheduled-tasks',
          name: 'scheduled-tasks',
          component: ScheduledTasksView,
        },
        { path: '/settings', name: 'settings', component: SettingsView },
      ],
    },
  ],
})

export default router
