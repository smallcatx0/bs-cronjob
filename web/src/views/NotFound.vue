<template>
  <el-result icon="warning" title="404" sub-title="抱歉，页面不存在">
    <template #extra>
      <el-button type="primary" @click="$router.push('/admin/jobs')">返回首页</el-button>
    </template>
  </el-result>
</template>

<script setup>
</script>
