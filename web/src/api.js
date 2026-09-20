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
