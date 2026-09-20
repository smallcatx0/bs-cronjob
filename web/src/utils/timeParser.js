
export function fmtTime(t) {
  return t ? String(t).replace('T', ' ').slice(0, 19) : '-' 
}

// 将秒数格式化为更易读的时长(天/小时/分钟/秒), 最多保留两个非零单位
export function fmtTtl(s) {
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