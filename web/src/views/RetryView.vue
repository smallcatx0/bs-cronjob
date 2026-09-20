<template>
  <div>
    <!-- 搜索栏 -->
    <el-form :inline="true" :model="query" @submit.prevent>
      <el-form-item label="Unkey">
        <el-input v-model="query.unkey" placeholder="任务唯一名" clearable style="width: 160px" @keyup.enter="load(1)" />
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
      </el-form-item>
    </el-form>

    <!-- 策略表格 -->
    <el-table :data="rows" v-loading="loading" border stripe size="small">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="unkey" label="Unkey" min-width="140" show-overflow-tooltip />
      <el-table-column prop="desc" label="描述" min-width="110" show-overflow-tooltip />
      <el-table-column prop="db_name" label="库名" min-width="100" show-overflow-tooltip />
      <el-table-column prop="table_name" label="表名" min-width="120" show-overflow-tooltip />
      <el-table-column label="依据字段" min-width="130" show-overflow-tooltip>
        <template #default="{ row }">{{ row.column_name }} ({{ row.column_type }})</template>
      </el-table-column>
      <el-table-column label="时间窗口" width="150">
        <template #default="{ row }">前 {{ row.before }}s / 跨度 {{ row.duration }}s</template>
      </el-table-column>
      <el-table-column label="更新字段" min-width="130" show-overflow-tooltip>
        <template #default="{ row }">{{ row.set_fields }}</template>
      </el-table-column>
      <el-table-column label="附加条件" min-width="130" show-overflow-tooltip>
        <template #default="{ row }">{{ row.find_wh }}</template>
      </el-table-column>
      <el-table-column prop="limit" label="批次" width="70" />
      <el-table-column prop="spec" label="调度规则" min-width="100" show-overflow-tooltip />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag>
        </template>
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
    <el-dialog v-model="editVisible" :title="form.id ? '编辑 Retry 策略' : '新建 Retry 策略'" width="600px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="Unkey" required>
          <el-input v-model="form.unkey" maxlength="128" :disabled="!!form.id" placeholder="任务唯一名, 创建后不可修改" />
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
        <el-form-item label="查找条件" required>
          <el-input v-model="form.find_wh" type="textarea" :rows="2" placeholder="时间窗口之外追加的 WHERE 条件, 如: status = 0" />
          <span class="tip">与时间窗口条件 AND 拼接, 不要重复写时间条件</span>
        </el-form-item>
        <el-form-item label="更新字段" required>
          <el-input v-model="form.set_fields" placeholder="UPDATE 的 SET 片段, 如: status = 1" />
        </el-form-item>
        <el-form-item label="窗口起点(秒)" required>
          <el-input-number v-model="form.before" :min="1" />
          <span class="tip">从当前时间往前 {{ form.before || 0 }} 秒作为窗口起点</span>
        </el-form-item>
        <el-form-item label="窗口跨度(秒)" required>
          <el-input-number v-model="form.duration" :min="1" />
          <span class="tip">窗口为 [now-before, now-before+duration)</span>
        </el-form-item>
        <el-form-item label="单次处理条数" required>
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
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listRetry, addRetry, updateRetry, deleteRetry, toggleRetry } from '../api'

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const page = reactive({ page: 1, limit: 20, total: 0 })
const query = reactive({ unkey: '', db_name: '', table_name: '', status: '' })

const editVisible = ref(false)
const form = reactive({ id: null, unkey: '', dsn: '', db_name: '', table_name: '', column_name: '', column_type: 'datetime', find_wh: '', set_fields: '', before: 300, duration: 60, limit: 1000, spec: '', desc: '' })

// cron 快捷输入(与 asynq 一致的标准 5 段: 分 时 日 月 周)
const quickCrons = [
  { label: '每分钟', expr: '* * * * *' },
  { label: '两分钟', expr: '*/2 * * * *' },
  { label: '五分钟', expr: '*/5 * * * *' },
  { label: '半小时', expr: '*/30 * * * *' },
]

async function load(p) {
  if (p) page.page = p
  loading.value = true
  try {
    const params = { page: page.page, limit: page.limit }
    for (const k of ['unkey', 'db_name', 'table_name', 'status']) if (query[k]) params[k] = query[k]
    const data = await listRetry(params)
    rows.value = data?.list || []
    Object.assign(page, data?.page || {})
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  Object.assign(form, row
    ? { id: row.id, unkey: row.unkey, dsn: row.dsn, db_name: row.db_name, table_name: row.table_name, column_name: row.column_name, column_type: row.column_type, find_wh: row.find_wh, set_fields: row.set_fields, before: row.before, duration: row.duration, limit: row.limit, spec: row.spec, desc: row.desc || '' }
    : { id: null, unkey: '', dsn: '', db_name: '', table_name: '', column_name: '', column_type: 'datetime', find_wh: '', set_fields: '', before: 300, duration: 60, limit: 1000, spec: '', desc: '' })
  editVisible.value = true
}

async function doSave() {
  saving.value = true
  try {
    if (form.id) {
      // 后端仅支持更新这些字段, unkey/spec 不可变更
      await updateRetry({
        id: form.id,
        dsn: form.dsn,
        db_name: form.db_name,
        table_name: form.table_name,
        column_name: form.column_name,
        column_type: form.column_type,
        find_wh: form.find_wh,
        set_fields: form.set_fields,
        before: form.before,
        duration: form.duration,
        limit: form.limit,
        desc: form.desc,
      })
    } else {
      await addRetry({
        unkey: form.unkey,
        dsn: form.dsn,
        db_name: form.db_name,
        table_name: form.table_name,
        column_name: form.column_name,
        column_type: form.column_type,
        find_wh: form.find_wh,
        set_fields: form.set_fields,
        before: form.before,
        duration: form.duration,
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
  await toggleRetry(row.id, status)
  ElMessage.success(status === 'online' ? '已上线' : '已下线')
  load()
}

async function doDelete(row) {
  await ElMessageBox.confirm(`确认删除策略「${row.unkey}」?`, '提示', { type: 'warning' })
  await deleteRetry(row.id)
  ElMessage.success('已删除')
  load()
}

onMounted(() => load(1))
</script>

<style scoped>
.tip { margin-left: 10px; font-size: 12px; color: #909399; }
.cron-quick { width: 100%; display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
.cron-tag { cursor: pointer; }
.cron-tag:hover { color: var(--el-color-primary); border-color: var(--el-color-primary); }
</style>