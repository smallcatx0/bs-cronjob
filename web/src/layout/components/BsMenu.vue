<template>
  <el-menu class="menu" :default-active="route.path" :collapse="isCollapse" router>
    <template v-for="item in authMenu" :key="item.path">
      <el-menu-item :index="item.path">
        <el-icon><component :is="item.icon" /></el-icon>
        <template #title>{{ item.label }}</template>
      </el-menu-item>
    </template>
  </el-menu>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { Timer, Document } from '@element-plus/icons-vue'

const route = useRoute()
const isCollapse = ref(false)

const authMenu = [
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
]
</script>

<style scoped>
.menu:not(.el-menu--collapse) {
  width: 200px;
}
.menu {
  min-height: 100%;
  border-right: none;
}
</style>
