// 将标准 5 段 cron 表达式(分 时 日 月 周)转化为友好阅读格式

export const WEEK_CN = { 0: '周日', 1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五', 6: '周六', 7: '周日' }

// 解析单个字段:返回 {type:'any'} | {type:'step',step} | {type:'list',values}(超范围或不支持的写法返回 null)
function parseCronField(field, min, max) {
  if (field === '*') return { type: 'any' }
  const stepMatch = field.match(/^(\*|\d+-\d+)\/(\d+)$/)
  if (stepMatch) {
    const step = Number(stepMatch[2])
    if (stepMatch[1] === '*') return { type: 'step', step }
    const [lo, hi] = stepMatch[1].split('-').map(Number)
    const values = []
    for (let i = lo; i <= hi; i += step) values.push(i)
    return { type: 'list', values }
  }
  const values = []
  for (const token of field.split(',')) {
    if (/^\d+$/.test(token)) {
      values.push(Number(token))
    } else if (/^\d+-\d+$/.test(token)) {
      const [lo, hi] = token.split('-').map(Number)
      for (let i = lo; i <= hi; i++) values.push(i)
    } else {
      return null
    }
  }
  if (!values.length || values.some((v) => v < min || v > max)) return null
  return { type: 'list', values: [...new Set(values)].sort((a, b) => a - b) }
}

// 将连续数字列表折叠为区间展示,如 [1,2,3,4,5] -> "1~5"
function foldCronList(vals, fmt) {
  const out = []
  let start = vals[0]
  let prev = vals[0]
  for (let i = 1; i <= vals.length; i++) {
    if (i < vals.length && vals[i] === prev + 1) {
      prev = vals[i]
      continue
    }
    if (start === prev) out.push(fmt(start))
    else if (prev === start + 1) out.push(fmt(start), fmt(prev))
    else out.push(`${fmt(start)}~${fmt(prev)}`)
    if (i < vals.length) {
      start = vals[i]
      prev = vals[i]
    }
  }
  return out.join('、')
}

export function cronToText(expr) {
  if (!expr) return '-'
  const parts = String(expr).trim().split(/\s+/)
  if (parts.length !== 5) return expr
  const minute = parseCronField(parts[0], 0, 59)
  const hour = parseCronField(parts[1], 0, 23)
  const dom = parseCronField(parts[2], 1, 31)
  const month = parseCronField(parts[3], 1, 12)
  const dow = parseCronField(parts[4], 0, 7)
  if (!minute || !hour || !dom || !month || !dow) return expr

  const pad = (n) => String(n).padStart(2, '0')
  const isAny = (f) => f.type === 'any'

  // 纯分钟级(时/日/月/周均为任意)
  if (isAny(hour) && isAny(dom) && isAny(month) && isAny(dow)) {
    if (minute.type === 'step') return `每${minute.step}分钟`
    if (isAny(minute)) return '每分钟'
    if (minute.type === 'list') return `每小时第 ${foldCronList(minute.values, (v) => v)} 分钟`
  }

  // 日期范围(月 / 日或周)
  const scope = []
  if (!isAny(month)) scope.push(foldCronList(month.values, (v) => `${v}月`))
  if (!isAny(dow)) scope.push(foldCronList(dow.values.map((v) => (v === 7 ? 0 : v)), (v) => WEEK_CN[v]))
  else if (!isAny(dom)) scope.push(dom.type === 'step' ? `每${dom.step}天` : foldCronList(dom.values, (v) => `${v}日`))

  // 时间(时 / 分)
  let time = ''
  if (!isAny(hour) && !isAny(minute)) {
    time = `${foldCronList(hour.values, pad)}:${foldCronList(minute.values, pad)}`
  } else if (!isAny(hour) && isAny(minute)) {
    time = `${foldCronList(hour.values, pad)}点每分钟`
  } else if (isAny(hour) && !isAny(minute)) {
    time = minute.type === 'step' ? `每${minute.step}分钟` : `每小时第${foldCronList(minute.values, (v) => v)}分钟`
  } else {
    time = '每分钟'
  }

  const head = scope.length ? scope.join(' ') : '每天'
  return `${head} ${time}`
}
