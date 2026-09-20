<template>
  <div>
    <!-- 搜索栏 -->
    <el-form :inline="true" :model="query" @submit.prevent>
      <el-form-item label="类型">
        <el-select v-model="query.kind" clearable placeholder="全部" style="width: 110px">
          <el-option label="ttl" value="ttl" />
          <el-option label="retry" value="retry" />
        </el-select>
      </el-form-item>
      <el-form-item label="策略ID">
        <el-input v-model.number="query.strategy_id" placeholder="strategy_id" clearable style="width: 110px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="结果">
        <el-select v-model="query.status" clearable placeholder="全部" style="width: 110px">
          <el-option label="running" value="running" />
          <el-option label="success" value="success" />
          <el-option label="failed" value="failed" />
        </el-select>
      </el-form-item>
      <el-form-item label="开始时间">
        <el-date-picker
          v-model="range"
          type="datetimerange"
          value-format="YYYY-MM-DD HH:mm:ss"
          start-placeholder="起"
          end-placeholder="止"
          style="width: 340px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="load(1)">查询</el-button>
        <el-button @click="load()">刷新</el-button>
      </el-form-item>
    </el-form>

    <!-- 日志表格(可展开行直接显示错误/输出) -->
    <el-table :data="rows" v-loading="loading" border stripe size="small">
      <el-table-column type="expand">
        <template #default="{ row }">
          <div style="padding: 8px 16px">
            <template v-if="row.error">
              <h4 style="color: #f56c6c; margin: 4px 0">错误</h4>
              <pre class="log-box err">{{ row.error }}</pre>
            </template>
            <h4 style="margin: 4px 0">输出</h4>
            <pre class="log-box">{{ row.output || '(无输出)' }}</pre>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="类型" width="80">
        <template #default="{ row }">
          <el-tag size="small" :type="row.kind === 'ttl' ? 'primary' : 'warning'">{{ row.kind }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="strategy_id" label="策略ID" width="80" />
      <el-table-column prop="strategy_name" label="策略名" min-width="150" show-overflow-tooltip />
      <el-table-column label="结果" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="{ running: 'info', success: 'success', failed: 'danger' }[row.status]">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="开始时间" width="160"><template #default="{ row }">{{ fmtTime(row.started_at) }}</template></el-table-column>
      <el-table-column label="结束时间" width="160"><template #default="{ row }">{{ fmtTime(row.finished_at) }}</template></el-table-column>
      <el-table-column label="耗时" width="100"><template #default="{ row }">{{ fmtCost(row) }}</template></el-table-column>
    </el-table>

    <el-pagination
      style="margin-top: 12px; justify-content: flex-end"
      layout="total, sizes, prev, pager, next"
      :total="page.total"
      v-model:current-page="page.page"
      v-model:page-size="page.limit"
      :page-sizes="[10, 20, 50, 100]"
      @change="load()"
    />
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { listStrategyLogs } from '../api'

const loading = ref(false)
const rows = ref([])
const page = reactive({ page: 1, limit: 20, total: 0 })
const query = reactive({ kind: '', strategy_id: '', status: '' })
const range = ref(null)

const fmtTime = (t) => (t ? String(t).replace('T', ' ').slice(0, 19) : '-')
const fmtCost = (row) => {
  if (!row.started_at || !row.finished_at) return '-'
  const ms = new Date(String(row.finished_at).replace('T', ' ').replace(/-/g, '/')
    .slice(0, 19)) - new Date(String(row.started_at).replace('T', ' ').replace(/-/g, '/')
    .slice(0, 19))
  return ms >= 0 ? ms + 'ms' : '-'
}

async function load(p) {
  if (p) page.page = p
  loading.value = true
  try {
    const params = { page: page.page, limit: page.limit }
    for (const k of ['kind', 'strategy_id', 'status']) if (query[k] !== '' && query[k] != null) params[k] = query[k]
    if (range.value?.length === 2) {
      params.start = range.value[0]
      params.end = range.value[1]
    }
    const data = await listStrategyLogs(params)
    rows.value = data?.list || []
    Object.assign(page, data?.page || {})
  } finally {
    loading.value = false
  }
}

onMounted(() => load(1))
</script>

<style scoped>
.log-box {
  background: #0b1021; color: #d5e0ff; padding: 10px; border-radius: 4px;
  max-height: 320px; overflow: auto; white-space: pre-wrap; word-break: break-all;
  font-size: 12px; margin: 4px 0;
}
.log-box.err { color: #ffb4b4; }
</style>
