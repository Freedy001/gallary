import {createRouter, createWebHistory, type RouteRecordRaw} from 'vue-router'
import {useAuthStore} from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/gallery',
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: {requiresAuth: false},
  },
  {
    path: '/gallery',
    name: 'Gallery',
    component: () => import('@/views/Gallery.vue'),
    meta: {requiresAuth: true},
  },
  {
    path: '/gallery/location',
    name: 'Location',
    component: () => import('@/views/LocationView.vue'),
    meta: {requiresAuth: true},
  },
  {
    path: '/gallery/share',
    name: 'ShareManage',
    component: () => import('@/views/ShareManage.vue'),
    meta: {requiresAuth: true},
  },
  {
    path: '/gallery/albums',
    name: 'Albums',
    component: () => import('@/views/AlbumsView.vue'),
    meta: {requiresAuth: true},
  },
  {
    path: '/gallery/albums/:id',
    name: 'AlbumDetail',
    component: () => import('@/views/AlbumDetailView.vue'),
    meta: {requiresAuth: true},
  },
  {
    path: '/s/:code',
    name: 'SharePublic',
    component: () => import('@/views/SharePublic.vue'),
    meta: {requiresAuth: false},
  },
  {
    path: '/gallery/people',
    name: 'People',
    component: () => import('@/views/PeopleView.vue'),
    meta: {requiresAuth: true, disabled: true},
  },
  {
    path: '/gallery/timeline',
    name: 'Timeline',
    component: () => import('@/views/TimelineView.vue'),
    meta: {requiresAuth: true, disabled: true},
  },
  {
    path: '/gallery/trash',
    name: 'Trash',
    component: () => import('@/views/TrashView.vue'),
    meta: {requiresAuth: true},
  },
  {
    path: '/gallery/settings',
    name: 'Settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: {requiresAuth: false},
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：检查认证状态
router.beforeEach(async (to, _, next) => {
  const authStore = useAuthStore()

  if (await authStore.requiresAuth()) {
    // 需要认证的路由
    if (to.meta.requiresAuth) {
      next('/login')
    } else {
      next()
    }
    return
  }

  // 如果不需要认证，直接放行
  if (to.path === '/login') {
    next('/gallery')
  } else {
    next()
  }
})

export default router
