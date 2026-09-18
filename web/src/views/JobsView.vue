<template>
  <div>
    <!-- 搜索栏 -->
    <el-form :inline="true" :model="query" @submit.prevent>
      <el-form-item label="名称">
        <el-input v-model="query.name" placeholder="任务名" clearable style="width: 160px" @keyup.enter="load(1)" />
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="query.type" clearable placeholder="全部" style="width: 120px">
          <el-option label="http" value="http" />
          <el-option label="shell" value="shell" />
          <el-option label="gofunc" value="gofunc" />
        </el-select>
      </el-form-item>
      <el-form-item label="调度">
        <el-select v-model="query.schedule_type" clearable placeholder="全部" style="width: 120px">
          <el-option label="cron" value="cron" />
          <el-option label="once" value="once" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="query.status" clearable placeholder="全部" style="width: 120px">
          <el-option label="停用" :value="0" />
          <el-option label="启用" :value="1" />
          <el-option label="已过期" :value="2" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="load(1)">查询</el-button>
        <el-button type="success" @click="openEdit(null)">新建任务</el-button>
      </el-form-item>
    </el-form>

    <!-- 任务表格 -->
    <el-table :data="rows" v-loading="loading" border stripe size="small">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" min-width="140" show-overflow-tooltip />
      <el-table-column prop="type" label="类型" width="80">
        <template #default="{ row }"><el-tag size="small">{{ row.type }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="schedule_type" label="调度" width="70" />
      <el-table-column label="调度规则" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.schedule_type === 'cron'">{{ row.cron_expr }}</span>
          <span v-else>{{ fmtTime(row.execute_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="下次运行" width="160">
        <template #default="{ row }">{{ fmtTime(row.next_run) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="doRun(row)">运行</el-button>
          <el-button size="small" v-if="row.status !== 1" type="success" :disabled="row.status === 2" @click="doToggle(row, 1)">启用</el-button>
          <el-button size="small" v-else type="warning" @click="doToggle(row, 0)">停用</el-button>
          <el-button size="small" :disabled="row.status === 1" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" :disabled="row.status === 1" @click="doDelete(row)">删除</el-button>
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
    <el-dialog v-model="editVisible" :title="form.id ? '编辑任务' : '新建任务'" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="128" />
        </el-form-item>
        <el-form-item label="任务类型" required>
          <el-radio-group v-model="form.type" :disabled="!!form.id">
            <el-radio-button value="http">http</el-radio-button>
            <el-radio-button value="shell">shell</el-radio-button>
            <el-radio-button value="gofunc">gofunc</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="调度类型" required>
          <el-radio-group v-model="form.schedule_type" :disabled="!!form.id">
            <el-radio-button value="cron">cron 周期</el-radio-button>
            <el-radio-button value="once">once 一次性</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.schedule_type === 'cron'" label="cron 表达式" required>
          <el-input v-model="form.cron_expr" placeholder="如: 0 */5 * * * ? (秒可选)" />
        </el-form-item>
        <el-form-item v-else label="执行时间" required>
          <el-date-picker v-model="form.execute_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" placeholder="选择执行时间" />
        </el-form-item>

        <!-- payload 按类型给出模板 -->
        <el-form-item v-if="form.type === 'http'" label="HTTP 配置">
          <el-input v-model="httpForm.method" style="width: 110px" placeholder="GET" />
          <el-input v-model="httpForm.url" style="width: calc(100% - 120px); margin-left: 8px" placeholder="http://..." />
          <el-input v-model="httpForm.body" type="textarea" :rows="2" style="margin-top: 6px" placeholder='请求体(可选), 及headers: {"headers":{"Token":"x"},"body":"","expect_status":200}' />
        </el-form-item>
        <el-form-item v-else-if="form.type === 'shell'" label="Shell 配置">
          <el-input v-model="shellForm.cmd" placeholder="白名单内脚本或程序, 如 /data/scripts/backup.sh" />
          <el-input v-model="shellForm.args" style="margin-top: 6px" placeholder='参数, 逗号分隔(可选)' />
        </el-form-item>
        <el-form-item v-else label="Go Func">
          <el-select v-model="gofuncForm.func" filterable allow-create placeholder="选择或输入已注册函数名" style="width: 100%">
            <el-option v-for="f in gofuncs" :key="f" :label="f" :value="f" />
          </el-select>
          <el-input v-model="gofuncForm.args" type="textarea" :rows="2" style="margin-top: 6px" placeholder='args JSON(可选), 如 {"name":"x"}' />
        </el-form-item>

        <el-form-item label="超时(秒)">
          <el-input-number v-model="form.timeout_sec" :min="1" :max="86400" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listJobs, addJob, updateJob, deleteJob, runJob, toggleJob, listGoFuncs } from '../api'

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const page = reactive({ page: 1, limit: 10, total: 0 })
const query = reactive({ name: '', type: '', schedule_type: '', status: null })

const editVisible = ref(false)
const form = reactive({ id: null, name: '', type: 'http', schedule_type: 'cron', cron_expr: '', execute_at: '', timeout_sec: 300 })
const httpForm = reactive({ method: 'GET', url: '', body: '' })
const shellForm = reactive({ cmd: '', args: '' })
const gofuncForm = reactive({ func: '', args: '' })
const gofuncs = ref([])

const fmtTime = (t) => (t ? String(t).replace('T', ' ').slice(0, 19) : '-')
const statusText = (s) => ({ 0: '停用', 1: '启用', 2: '已过期' }[s] ?? s)
const statusTag = (s) => ({ 0: 'info', 1: 'success', 2: 'warning' }[s] ?? 'info')

async function load(p) {
  if (p) page.page = p
  loading.value = true
  try {
    const params = { page: page.page, limit: page.limit }
    for (const k of ['name', 'type', 'schedule_type']) if (query[k]) params[k] = query[k]
    if (query.status !== null && query.status !== '') params.status = query.status
    const data = await listJobs(params)
    rows.value = data?.list || []
    Object.assign(page, data?.page || {})
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  Object.assign(form, row
    ? { id: row.id, name: row.name, type: row.type, schedule_type: row.schedule_type, cron_expr: row.cron_expr, execute_at: fmtTime(row.execute_at) === '-' ? '' : fmtTime(row.execute_at), timeout_sec: row.timeout_sec }
    : { id: null, name: '', type: 'http', schedule_type: 'cron', cron_expr: '', execute_at: '', timeout_sec: 300 })
  httpForm.method = 'GET'; httpForm.url = ''; httpForm.body = ''
  shellForm.cmd = ''; shellForm.args = ''
  gofuncForm.func = ''; gofuncForm.args = ''
  if (row?.payload) {
    try {
      const p = JSON.parse(row.payload)
      if (row.type === 'http') Object.assign(httpForm, { method: p.method || 'GET', url: p.url || '', body: p.body || '' })
      if (row.type === 'shell') Object.assign(shellForm, { cmd: p.cmd || '', args: (p.args || []).join(',') })
      if (row.type === 'gofunc') Object.assign(gofuncForm, { func: p.func || '', args: p.args ? JSON.stringify(p.args) : '' })
    } catch { /* ignore */ }
  }
  listGoFuncs().then((d) => (gofuncs.value = d || [])).catch(() => {})
  editVisible.value = true
}

function buildPayload() {
  if (form.type === 'http') {
    const p = { method: httpForm.method || 'GET', url: httpForm.url }
    if (httpForm.body) {
      try { Object.assign(p, JSON.parse(httpForm.body)) } catch { p.body = httpForm.body }
    }
    return JSON.stringify(p)
  }
  if (form.type === 'shell') {
    return JSON.stringify({ cmd: shellForm.cmd, args: shellForm.args ? shellForm.args.split(',').map((s) => s.trim()).filter(Boolean) : [] })
  }
  const p = { func: gofuncForm.func }
  if (gofuncForm.args) { try { p.args = JSON.parse(gofuncForm.args) } catch { throw new Error('gofunc args 不是合法JSON') } }
  return JSON.stringify(p)
}

async function doSave() {
  let payload
  try { payload = buildPayload() } catch (e) { ElMessage.error(e.message); return }
  const body = { name: form.name, type: form.type, schedule_type: form.schedule_type, cron_expr: form.cron_expr, execute_at: form.execute_at, payload, timeout_sec: form.timeout_sec }
  saving.value = true
  try {
    if (form.id) await updateJob({ id: form.id, ...body })
    else await addJob(body)
    ElMessage.success('保存成功')
    editVisible.value = false
    load()
  } finally {
    saving.value = false
  }
}

async function doRun(row) {
  await runJob(row.id)
  ElMessage.success('已触发执行, 请到「运行记录」查看结果')
}

async function doToggle(row, status) {
  await toggleJob(row.id, status)
  ElMessage.success(status === 1 ? '已启用' : '已停用')
  load()
}

async function doDelete(row) {
  await ElMessageBox.confirm(`确认删除任务「${row.name}」?`, '提示', { type: 'warning' })
  await deleteJob(row.id)
  ElMessage.success('已删除')
  load()
}

onMounted(() => load(1))
</script>
