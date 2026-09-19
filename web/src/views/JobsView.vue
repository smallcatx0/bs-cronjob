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
      <el-table-column label="内容" width="80">
        <template #default="{ row }"><el-button size="small" link type="primary" @click="showContent(row)">查看</el-button></template>
      </el-table-column>
      <el-table-column label="日志" width="80">
        <template #default="{ row }"><el-button size="small" link type="primary" @click="showLogs(row)">查看</el-button></template>
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
          <el-input v-model="form.name" maxlength="128" :disabled="!!form.id" />
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
          <div class="cron-quick">
            <el-tag v-for="c in quickCrons" :key="c.label" size="small" effect="plain" class="cron-tag" @click="form.cron_expr = c.expr">{{ c.label }}</el-tag>
          </div>
          <el-input v-model="form.cron_expr" placeholder="如: */5 * * * * (5段: 分 时 日 月 周)" />
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

    <!-- 内容详情弹窗 -->
    <el-dialog v-model="contentVisible" :title="contentTitle" width="640px">
      <el-tree v-if="payloadTree.length" :data="payloadTree" default-expand-all :expand-on-click-node="false">
        <template #default="{ data }">
          <span class="json-node">
            <span class="json-key">{{ data.label }}</span>
            <span v-if="'value' in data" class="json-val">{{ formatVal(data.value) }}</span>
          </span>
        </template>
      </el-tree>
      <el-empty v-else description="无内容" />
      <template #footer>
        <el-button @click="contentVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 执行日志弹窗 -->
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
        <el-table-column prop="trigger_type" label="触发" width="80" />
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="{ running: 'info', success: 'success', failed: 'danger' }[row.status]">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="160"><template #default="{ row }">{{ fmtTime(row.started_at) }}</template></el-table-column>
        <el-table-column label="结束时间" width="160"><template #default="{ row }">{{ fmtTime(row.finished_at) }}</template></el-table-column>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { listJobs, addJob, updateJob, deleteJob, runJob, toggleJob, listGoFuncs, listLogs } from '../api'

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const page = reactive({ page: 1, limit: 20, total: 0 })
const query = reactive({ name: '', type: '', schedule_type: '', status: null })

const editVisible = ref(false)
const form = reactive({ id: null, name: '', type: 'http', schedule_type: 'cron', cron_expr: '', execute_at: '', timeout_sec: 300 })
const httpForm = reactive({ method: 'GET', url: '', body: '' })
const shellForm = reactive({ cmd: '', args: '' })
const gofuncForm = reactive({ func: '', args: '' })
const gofuncs = ref([])

// cron 快捷输入(与 asynq 一致的标准 5 段: 分 时 日 月 周)
const quickCrons = [
  { label: '每分钟', expr: '* * * * *' },
  { label: '两分钟', expr: '*/2 * * * *' },
  { label: '五分钟', expr: '*/5 * * * *' },
  { label: '半小时', expr: '*/30 * * * *' },
]

const contentVisible = ref(false)
const contentTitle = ref('任务内容')
const payloadTree = ref([])

// 执行日志弹窗
const logsVisible = ref(false)
const logsTitle = ref('执行日志')
const logsLoading = ref(false)
const logRows = ref([])
const logsPage = reactive({ page: 1, limit: 10, total: 0 })
const currentJobId = ref(null)

// 将任意 JSON 转为 el-tree 可展开的节点树
function buildTree(obj) {
  return Object.entries(obj).map(([k, v]) => {
    const node = { label: k }
    if (v !== null && typeof v === 'object') {
      node.children = Array.isArray(v)
        ? v.map((item, i) => (item !== null && typeof item === 'object'
            ? { label: `[${i}]`, children: buildTree(item) }
            : { label: `[${i}]`, value: item }))
        : buildTree(v)
    } else {
      node.value = v
    }
    return node
  })
}

const formatVal = (v) => (typeof v === 'string' ? `"${v}"` : String(v))

function showContent(row) {
  contentTitle.value = `任务内容 - ${row.name}`
  try {
    const parsed = JSON.parse(row.payload || '{}')
    payloadTree.value = (parsed && typeof parsed === 'object') ? buildTree(parsed) : []
  } catch {
    payloadTree.value = []
  }
  contentVisible.value = true
}

async function loadLogs() {
  if (!currentJobId.value) return
  logsLoading.value = true
  try {
    const data = await listLogs({ job_id: currentJobId.value, page: logsPage.page, limit: logsPage.limit })
    logRows.value = data?.list || []
    Object.assign(logsPage, data?.page || {})
  } finally {
    logsLoading.value = false
  }
}

function showLogs(row) {
  currentJobId.value = row.id
  logsTitle.value = `执行日志 - ${row.name}`
  logsPage.page = 1
  logsPage.total = 0
  logRows.value = []
  logsVisible.value = true
  loadLogs()
}

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

<style scoped>
.cron-quick { width: 100%; display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
.cron-tag { cursor: pointer; }
.cron-tag:hover { color: var(--el-color-primary); border-color: var(--el-color-primary); }
.json-node { display: inline-flex; gap: 6px; font-size: 13px; }
.json-key { color: #7d3ff5; font-weight: 600; }
.json-val { color: #1f7a3f; word-break: break-all; }
.log-box {
  background: #0b1021; color: #d5e0ff; padding: 10px; border-radius: 4px;
  max-height: 320px; overflow: auto; white-space: pre-wrap; word-break: break-all;
  font-size: 12px; margin: 4px 0;
}
.log-box.err { color: #ffb4b4; }
</style>
