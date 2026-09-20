import { Timer, Document, DataAnalysis } from '@element-plus/icons-vue'

// 侧边菜单栏配置，后续新增/调整菜单直接修改此文件即可
export const authMenu = [
  {
    label: '任务管理',
    path: '/admin/jobs',
    icon: Timer,
  },
  {
    label: '运行记录',
    path: '/admin/logs',
    icon: Document,
  },
  {
    label: 'TTL策略',
    path: '/admin/ttl',
    icon: Document,
  },
  {
    label: 'Retry策略',
    path: '/admin/retry',
    icon: Document,
  },
  {
    label: '策略日志',
    path: '/admin/strategy-logs',
    icon: DataAnalysis,
  },
]
