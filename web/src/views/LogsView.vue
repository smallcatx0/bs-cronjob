<template>
  <div>
    <el-form :inline="true" :model="query" @submit.prevent>
      <el-form-item label="任务ID">
        <el-input v-model.number="query.job_id" placeholder="job_id" clearable style="width: 110px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="任务名">
        <el-input v-model="query.job_name" clearable style="width: 150px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="触发">
        <el-select v-model="query.trigger_type" clearable placeholder="全部" style="width: 110px">
          <el-option label="cron" value="cron" />
          <el-option label="once" value="once" />
          <el-option label="manual" value="manual" />
        </el-select>
      </el-form-item>
      <el-form-item label="结果">
        <el-select v-model="query.status" clearable placeholder="全部" style="width: 110px">
          <el-option label="running" value="running" />
          <el-option label="success" value="success" />
          <el-option label="failed" value="failed" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="load(1)">查询</el-button>
        <el-button @click="load()">刷新</el-button>
      </el-form-item>
    </el-form>

    <el-table :data="rows" v-loading="loading" border stripe size="small">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="job_id" label="任务ID" width="70" />
      <el-table-column prop="job_name" label="任务名" min-width="120" show-overflow-tooltip />
      <el-table-column prop="trigger_type" label="触发" width="80" />
      <el-table-column label="结果" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="{ running: 'info', success: 'success', failed: 'danger' }[row.status]">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="开始时间" width="160"><template #default="{ row }">{{ fmtTime(row.started_at) }}</template></el-table-column>
      <el-table-column label="结束时间" width="160"><template #default="{ row }">{{ fmtTime(row.finished_at) }}</template></el-table-column>
      <el-table-column label="详情" width="80">
        <template #default="{ row }"><el-button size="small" link type="primary" @click="showDetail(row)">查看</el-button></template>
      </el-table-column>
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

    <el-dialog v-model="detailVisible" title="运行详情" width="720px">
      <el-descriptions :column="2" border size="small" style="margin-bottom: 12px">
        <el-descriptions-item label="任务">{{ current.job_name }} (#{{ current.job_id }})</el-descriptions-item>
        <el-descriptions-item label="触发方式">{{ current.trigger_type }}</el-descriptions-item>
        <el-descriptions-item label="开始">{{ fmtTime(current.started_at) }}</el-descriptions-item>
        <el-descriptions-item label="结束">{{ fmtTime(current.finished_at) }}</el-descriptions-item>
      </el-descriptions>
      <template v-if="current.error">
        <h4 style="color: #f56c6c">错误</h4>
        <pre class="log-box err">{{ current.error }}</pre>
      </template>
      <h4>输出</h4>
      <pre class="log-box">{{ current.output || '(无输出)' }}</pre>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { listLogs } from '../api'

const loading = ref(false)
const rows = ref([])
const page = reactive({ page: 1, limit: 10, total: 0 })
const query = reactive({ job_id: '', job_name: '', status: '', trigger_type: '' })
const detailVisible = ref(false)
const current = ref({})

const fmtTime = (t) => (t ? String(t).replace('T', ' ').slice(0, 19) : '-')

async function load(p) {
  if (p) page.page = p
  loading.value = true
  try {
    const params = { page: page.page, limit: page.limit }
    for (const k of ['job_id', 'job_name', 'status', 'trigger_type']) if (query[k] !== '' && query[k] != null) params[k] = query[k]
    const data = await listLogs(params)
    rows.value = data?.list || []
    Object.assign(page, data?.page || {})
  } finally {
    loading.value = false
  }
}

function showDetail(row) {
  current.value = row
  detailVisible.value = true
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
