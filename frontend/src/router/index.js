import { createRouter, createWebHistory } from 'vue-router'
import routes from './routes'
import NProgress from 'nprogress'
import store from '@/state/store'

NProgress.configure({ showSpinner: false, trickleSpeed: 200 })

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior() {
    return { top: 0 }
  }
})

router.beforeEach((to, _from, next) => {
  NProgress.start()
  const user = store.getters['auth/user'] || (() => {
    try { return JSON.parse(sessionStorage.getItem('sc_user') || 'null') } catch { return null }
  })()
  const requiresAuth = to.matched.some(r => r.meta.requiresAuth !== false)
  const requiredRoles = to.matched
    .map(r => r.meta?.roles)
    .filter(Boolean)
    .flat()

  if (!user && requiresAuth && to.path !== '/login') {
    next('/login')
    return
  }

  if (user && to.path === '/login') {
    next('/dashboard')
    return
  }

  if (user && requiredRoles.length) {
    const role = user.role || 'user'
    if (!requiredRoles.includes(role)) {
      next('/access-denied')
      return
    }
  }

  document.title = to.meta.title ? `${to.meta.title} — SentinelCore` : 'SentinelCore'
  next()
})

router.afterEach(() => {
  NProgress.done()
})

export default router
