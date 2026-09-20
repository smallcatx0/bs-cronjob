import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/admin/jobs',
  },
  {
    path: '/admin',
    name: 'admin',
    component: () => import('~/layout/admin.vue'),
    children: [
      {
        path: '/admin',
        redirect: '/admin/jobs',
      },
      {
        path: '/admin/jobs',
        name: 'jobs',
        component: () => import('~/views/JobsView.vue'),
      },
      {
        path: '/admin/ttl',
        name: 'ttl',
        component: () => import('~/views/TtlView.vue'),
      },
      {
        path: '/admin/retry',
        name: 'retry',
        component: () => import('~/views/RetryView.vue'),
      },
    ],
  },
]

// 404 路由加载到最后
routes.push({
  path: '/:pathMatch(.*)*',
  name: 'NotFound',
  component: () => import('~/views/NotFound.vue'),
})

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
