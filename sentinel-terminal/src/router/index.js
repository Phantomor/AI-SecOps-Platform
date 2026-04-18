// src/router/index.js
import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../store'
import Home from '../views/Home.vue'
import ComplianceCopilot from '../views/ComplianceCopilot.vue'
import ThreatHunter from '../views/ThreatHunter.vue'
import LoginRegister from '../views/LoginRegister.vue'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: Home,
    meta: {
      title: 'ZeroTrust Sentinel - AI SecOps 平台',
      requiresAuth: false
    }
  },
  {
    path: '/login',
    name: 'LoginRegister',
    component: LoginRegister,
    meta: {
      title: 'SYSTEM_LOGIN // 身份验证',
      requiresAuth: false
    }
  },
  {
    path: '/compliance-copilot',
    name: 'ComplianceCopilot',
    component: ComplianceCopilot,
    meta: {
      title: 'Compliance Copilot | 合规审计',
      requiresAuth: true
    }
  },
  {
    path: '/threat-hunter',
    name: 'ThreatHunter',
    component: ThreatHunter,
    meta: {
      title: 'ThreatHunter | 威胁追捕',
      requiresAuth: true
    }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 全局路由守卫
router.beforeEach((to, from, next) => {
  // 设置页面标题
  if (to.meta.title) {
    document.title = to.meta.title
  }
  
  // 检查是否需要登录
  if (to.meta.requiresAuth) {
    const userStore = useUserStore()
    // 检查用户是否已登录
    if (!userStore.isLoggedIn) {
      // 未登录时，跳转到登录页面并传递终端风格的提示信息
      next({ name: 'LoginRegister', query: { msg: 'ERR_UNAUTHORIZED: 请先完成身份验证' } })
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router