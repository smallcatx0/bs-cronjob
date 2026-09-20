<template>
  <el-container class="main">
    <el-header class="header">
      <BsHeader />
    </el-header>
    <el-container>
      <el-aside style="width: auto;"><BsMenu /></el-aside>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import BsHeader from './components/BsHeader.vue'
import BsMenu from './components/BsMenu.vue'
</script>

<style scoped>
.main {
  background-color: #f3f4f5;
  min-height: 100vh;
}
.header {
  display: flex;
  align-items: center;
  background: #001529;
  color: #fff;
  padding-inline: 0;
}
.el-aside {
  background: #fff;
}
.el-main {
  padding-top: 8px;
}
</style>
