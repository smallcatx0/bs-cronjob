<template>
  <div>
    <!-- 搜索栏 -->
    <el-form :inline="true" :model="query" @submit.prevent>
      <el-form-item label="Unkey">
        <el-input v-model="query.unkey" placeholder="策略唯一key" clearable style="width: 160px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="库名">
        <el-input v-model="query.db_name" placeholder="数据库名" clearable style="width: 140px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="表名">
        <el-input v-model="query.table_name" placeholder="表名" clearable style="width: 160px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="query.status" clearable placeholder="全部" style="width: 120px">
          <el-option label="offline" value="offline" />
          <el-option label="online" value="online" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="load(1)">查询</el-button>
        <el-button type="success" @click="openEdit(null)">新建策略</el-button>
        <RouterLink to="/admin/strategy-logs"><el-button>策略日志</el-button></RouterLink>
      </el-form-item>
    </el-form>

    <!-- 策略表格 -->
    <el-table :data="rows" v-loading="loading" border stripe size="small">
      <el-table-column prop="id" label="#" width="60" />
      <el-table-column prop="unkey" label="名称" min-width="120" show-overflow-tooltip />
      <el-table-column prop="desc" label="描述" min-width="140" show-overflow-tooltip />
      <el-table-column prop="spec" label="调度规则" min-width="110" show-overflow-tooltip />
      <el-table-column label="库表" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.db_name ? row.db_name + '.' + row.table_name : row.table_name }}</template>
      </el-table-column>
      <el-table-column label="依据字段" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.column_name }} ({{ row.column_type }})</template>
      </el-table-column>
      <el-table-column label="TTL" width="130">
        <template #default="{ row }">
          <el-tooltip :content="`${row.ttl_value} 秒`" placement="top">
            <span>{{ fmtTtl(row.ttl_value) }}</span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column prop="limit" label="批次" width="80" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="日志" width="70">
        <template #default="{ row }"><el-button size="small" link type="primary" @click="showLogs(row)">查看</el-button></template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button size="small" v-if="row.status !== 'online'" type="success" @click="doToggle(row, 'online')">上线</el-button>
          <el-button size="small" v-else type="warning" @click="doToggle(row, 'offline')">下线</el-button>
          <el-button size="small" :disabled="row.status === 'online'" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" :disabled="row.status === 'online'" @click="doDelete(row)">删除</el-button>
        </template>
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

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="editVisible" :title="form.id ? '编辑 TTL 策略' : '新建 TTL 策略'" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="Unkey" required>
          <el-input v-model="form.unkey" maxlength="128" :disabled="!!form.id" placeholder="策略唯一key, 创建后不可修改" />
        </el-form-item>
        <el-form-item label="DSN" required>
          <el-input v-model="form.dsn" type="textarea" :rows="2" placeholder="如: user:pass@tcp(127.0.0.1:3306)/dbname?parseTime=true" />
        </el-form-item>
        <el-form-item label="数据库名">
          <el-input v-model="form.db_name" maxlength="64" placeholder="可选, 仅备注展示" />
        </el-form-item>
        <el-form-item label="表名" required>
          <el-input v-model="form.table_name" maxlength="128" :disabled="!!form.id" />
        </el-form-item>
        <el-form-item label="依据字段" required>
          <el-input v-model="form.column_name" maxlength="64" placeholder="如: created_at" />
        </el-form-item>
        <el-form-item label="字段类型" required>
          <el-radio-group v-model="form.column_type">
            <el-radio-button value="unix">unix</el-radio-button>
            <el-radio-button value="timestamp">timestamp</el-radio-button>
            <el-radio-button value="datetime">datetime</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="TTL(秒)" required>
          <el-input-number v-model="form.ttl_value" :min="1" />
          <span class="tip">超过该时长的数据将被清理, 单位秒</span>
        </el-form-item>
        <el-form-item label="单次删除条数" required>
          <el-input-number v-model="form.limit" :min="1" :max="100000" />
        </el-form-item>
        <el-form-item v-if="!form.id" label="调度规则" required>
          <div class="cron-quick">
            <el-tag v-for="c in quickCrons" :key="c.label" size="small" effect="plain" class="cron-tag" @click="form.spec = c.expr">{{ c.label }}</el-tag>
          </div>
          <el-input v-model="form.spec" placeholder="如: */5 * * * * (5段: 分 时 日 月 周)" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.desc" type="textarea" :rows="2" maxlength="255" show-word-limit placeholder="策略描述(可选)" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 执行日志弹窗(可展开行直接显示错误/输出, 沿用 JobsView 模式) -->
    <el-dialog v-model="logsVisible" :title="logsTitle" width="760px">
      <el-table :data="logRows" v-loading="logsLoading" border stripe size="small">
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
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="{ running: 'info', success: 'success', failed: 'danger' }[row.status]">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="160"><template #default="{ row }">{{ fmtTime(row.started_at) }}</template></el-table-column>
        <el-table-column label="结束时间" width="160"><template #default="{ row }">{{ fmtTime(row.finished_at) }}</template></el-table-column>
        <el-table-column label="耗时" width="90"><template #default="{ row }">{{ fmtCost(row) }}</template></el-table-column>
      </el-table>

      <el-pagination
        style="margin-top: 12px; justify-content: flex-end"
        layout="total, sizes, prev, pager, next"
        :total="logsPage.total"
        v-model:current-page="logsPage.page"
        v-model:page-size="logsPage.limit"
        :page-sizes="[10, 20, 50]"
        @change="loadLogs"
      />
      <template #footer>
        <el-button @click="logsVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listTtl, addTtl, updateTtl, deleteTtl, toggleTtl, listStrategyLogs } from '../api'

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const page = reactive({ page: 1, limit: 20, total: 0 })
const query = reactive({ unkey: '', db_name: '', table_name: '', status: '' })

const editVisible = ref(false)
const form = reactive({ id: null, unkey: '', dsn: '', db_name: '', table_name: '', column_name: '', column_type: 'datetime', ttl_value: 3600, limit: 1000, spec: '', desc: '' })
// 记录编辑时后端返回的脱敏 dsn, 用于判断用户是否修改了该字段
const originalDsn = ref('')

// 执行日志弹窗
const logsVisible = ref(false)
const logsTitle = ref('执行日志')
const logsLoading = ref(false)
const logRows = ref([])
const logsPage = reactive({ page: 1, limit: 10, total: 0 })
const currentStrategyId = ref(null)

const fmtTime = (t) => (t ? String(t).replace('T', ' ').slice(0, 19) : '-')
const fmtCost = (row) => {
  if (!row.started_at || !row.finished_at) return '-'
  const ms = new Date(String(row.finished_at).replace('T', ' ').replace(/-/g, '/')
    .slice(0, 19)) - new Date(String(row.started_at).replace('T', ' ').replace(/-/g, '/')
    .slice(0, 19))
  return ms >= 0 ? ms + 'ms' : '-'
}

// 将秒数格式化为更易读的时长(天/小时/分钟/秒), 最多保留两个非零单位
const fmtTtl = (s) => {
  const n = Number(s)
  if (!n || n <= 0) return '-'
  const units = [
    { label: '天', sec: 86400 },
    { label: '小时', sec: 3600 },
    { label: '分钟', sec: 60 },
    { label: '秒', sec: 1 },
  ]
  let rest = n
  const parts = []
  for (const u of units) {
    const v = Math.floor(rest / u.sec)
    if (v > 0) {
      parts.push(`${v}${u.label}`)
      rest -= v * u.sec
    }
    if (parts.length === 2) break
  }
  return parts.join('') || `${n}秒`
}

// cron 快捷输入(与 asynq 一致的标准 5 段: 分 时 日 月 周)
const quickCrons = [
  { label: '五分钟', expr: '*/5 * * * *' },
  { label: '半小时', expr: '*/30 * * * *' },
  { label: '一小时', expr: '0 * * * *' },
  { label: '每天', expr: '0 0 * * *' },
]

async function load(p) {
  if (p) page.page = p
  loading.value = true
  try {
    const params = { page: page.page, limit: page.limit }
    for (const k of ['unkey', 'db_name', 'table_name', 'status']) if (query[k]) params[k] = query[k]
    const data = await listTtl(params)
    rows.value = data?.list || []
    Object.assign(page, data?.page || {})
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  Object.assign(form, row
    ? { id: row.id, unkey: row.unkey, dsn: row.dsn, db_name: row.db_name, table_name: row.table_name, column_name: row.column_name, column_type: row.column_type, ttl_value: row.ttl_value, limit: row.limit, spec: row.spec, desc: row.desc || '' }
    : { id: null, unkey: '', dsn: '', db_name: '', table_name: '', column_name: '', column_type: 'datetime', ttl_value: 3600, limit: 1000, spec: '', desc: '' })
  originalDsn.value = row ? row.dsn : ''
  editVisible.value = true
}

async function doSave() {
  saving.value = true
  try {
    if (form.id) {
      // 后端仅支持更新这些字段, unkey/spec 不可变更
      const payload = {
        id: form.id,
        db_name: form.db_name,
        table_name: form.table_name,
        column_name: form.column_name,
        column_type: form.column_type,
        ttl_value: form.ttl_value,
        limit: form.limit,
        desc: form.desc,
      }
      // dsn 后端已脱敏, 仅在用户真实修改时才提交, 避免把脱敏串写回
      if (form.dsn !== originalDsn.value) {
        payload.dsn = form.dsn
      }
      await updateTtl(payload)
    } else {
      await addTtl({
        unkey: form.unkey,
        dsn: form.dsn,
        db_name: form.db_name,
        table_name: form.table_name,
        column_name: form.column_name,
        column_type: form.column_type,
        ttl_value: form.ttl_value,
        limit: form.limit,
        spec: form.spec,
        desc: form.desc,
      })
    }
    ElMessage.success('保存成功')
    editVisible.value = false
    load()
  } finally {
    saving.value = false
  }
}

async function doToggle(row, status) {
  await toggleTtl(row.id, status)
  ElMessage.success(status === 'online' ? '已上线' : '已下线')
  load()
}

async function doDelete(row) {
  await ElMessageBox.confirm(`确认删除策略「${row.unkey}」?`, '提示', { type: 'warning' })
  await deleteTtl(row.id)
  ElMessage.success('已删除')
  load()
}

async function loadLogs() {
  if (!currentStrategyId.value) return
  logsLoading.value = true
  try {
    const data = await listStrategyLogs({ kind: 'ttl', strategy_id: currentStrategyId.value, page: logsPage.page, limit: logsPage.limit })
    logRows.value = data?.list || []
    Object.assign(logsPage, data?.page || {})
  } finally {
    logsLoading.value = false
  }
}

function showLogs(row) {
  currentStrategyId.value = row.id
  logsTitle.value = `执行日志 - ${row.unkey}`
  logsPage.page = 1
  logsPage.total = 0
  logRows.value = []
  logsVisible.value = true
  loadLogs()
}

onMounted(() => load(1))
</script>

<style scoped>
.tip { margin-left: 10px; font-size: 12px; color: #909399; }
.cron-quick { width: 100%; display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
.cron-tag { cursor: pointer; }
.cron-tag:hover { color: var(--el-color-primary); border-color: var(--el-color-primary); }
.log-box {
  background: #0b1021; color: #d5e0ff; padding: 10px; border-radius: 4px;
  max-height: 320px; overflow: auto; white-space: pre-wrap; word-break: break-all;
  font-size: 12px; margin: 4px 0;
}
.log-box.err { color: #ffb4b4; }
</style>