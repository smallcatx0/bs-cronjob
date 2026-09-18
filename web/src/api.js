import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({ baseURL: '/admin/jobs', timeout: 15000 })

http.interceptors.response.use(
  (res) => {
    const b = res.data
    if (b.errcode !== 200) {
      ElMessage.error(b.msg || '请求失败')
      return Promise.reject(new Error(b.msg))
    }
    return b.data
  },
  (err) => {
    const msg = err.response?.data?.msg || err.message
    ElMessage.error(msg)
    return Promise.reject(err)
  },
)

export const listJobs = (params) => http.get('/list', { params })
export const jobDetail = (id) => http.get('/detail', { params: { id } })
export const addJob = (data) => http.post('/add', data)
export const updateJob = (data) => http.post('/update', data)
export const deleteJob = (id) => http.post('/delete', { id })
export const runJob = (id) => http.post('/run', { id })
export const toggleJob = (id, status) => http.post('/toggle', { id, status })
export const listLogs = (params) => http.post('/log?' + new URLSearchParams(params))
export const listGoFuncs = () => http.get('/gofuncs')
