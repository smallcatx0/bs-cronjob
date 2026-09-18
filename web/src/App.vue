<template>
  <el-container style="height: 100vh">
    <el-header style="display: flex; align-items: center; background: #001529; color: #fff">
      <h2 style="margin: 0; font-size: 18px">定时任务管理平台</h2>
    </el-header>
    <el-main>
      <el-tabs v-model="tab">
        <el-tab-pane label="任务管理" name="jobs"><JobsView /></el-tab-pane>
        <el-tab-pane label="运行记录" name="logs"><LogsView /></el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref } from 'vue'
import JobsView from './views/JobsView.vue'
import LogsView from './views/LogsView.vue'
const tab = ref('jobs')
</script>

<style>
body { margin: 0; font-family: -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif; }
.el-main { padding-top: 8px; }
</style>
