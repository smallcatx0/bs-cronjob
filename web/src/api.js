import { admin } from './request'

export const listJobs = (params) => admin.get('/jobs/list', { params })
export const jobDetail = (id) => admin.get('/jobs/detail', { params: { id } })
export const addJob = (data) => admin.post('/jobs/add', data)
export const updateJob = (data) => admin.post('/jobs/update', data)
export const deleteJob = (id) => admin.post('/jobs/delete', { id })
export const runJob = (id) => admin.post('/jobs/run', { id })
export const toggleJob = (id, status) => admin.post('/jobs/toggle', { id, status })
export const listLogs = (params) => admin.post('/jobs/log?' + new URLSearchParams(params))
export const listGoFuncs = () => admin.get('/jobs/gofuncs')

export const listTtl = (params) => admin.get('/tabledata/ttl/list', { params })
export const addTtl = (data) => admin.post('/tabledata/ttl/add', data)
export const updateTtl = (data) => admin.post('/tabledata/ttl/update', data)
export const deleteTtl = (id) => admin.post('/tabledata/ttl/delete', { id })
export const toggleTtl = (id, status) => admin.post('/tabledata/ttl/toggle', { id, status })

export const listRetry = (params) => admin.get('/tabledata/retry/list', { params })
export const addRetry = (data) => admin.post('/tabledata/retry/add', data)
export const updateRetry = (data) => admin.post('/tabledata/retry/update', data)
export const deleteRetry = (id) => admin.post('/tabledata/retry/delete', { id })
export const toggleRetry = (id, status) => admin.post('/tabledata/retry/toggle', { id, status })

// TTL/Retry 策略执行日志(kind/strategy_id/status/start/end + 分页)
export const listStrategyLogs = (params) => admin.get('/tabledata/ttl/log', { params })