
import axios from "axios"
import { ElNotification } from 'element-plus'

const admin = axios.create({ baseURL: '/admin', timeout: 15000 })

admin.interceptors.response.use(
  (res) => {
    const b = res.data
    if (b.errcode != 0) {
      ElNotification({ title: b.msg || '请求失败', type: 'error', duration: 0 })
      return Promise.reject(new Error(b.msg))
    }
    return b.data
  },
  (err) => {
    if (!err || !err.response) {
      ElNotification({
        title: "网络错误,请稍后重试",
        type: 'error',
        duration: 0,
      })
      return Promise.reject(err)
    }
    // 响应的拦截
    const httpCode = err.response.status
    switch (httpCode) {
      case 401:
        handel401()
        break;
      default:
        handelDefault(err.response)
        break;
    }
    return Promise.reject(err)
  },
)

function handel401() {
  // 删除cookie中的token
  delToken()
  // 提示
  ElNotification({
    title: "登录超时,请重新登录",
    type: 'error',
    duration: 0,
  })
  // 自动跳转到登录页面
  window.location.href = "/#"
}


function handelDefault(response) {
  let title, message
  if (response.data) {
    let { errcode, msg, request_id } = response.data
    title = msg
    message = `errcode=${errcode},request_id=${request_id}`
  } else {
    title = "网络错误,请稍后重试"
  }

  ElNotification({
    title,
    message,
    type: 'error',
    duration: 0,
  })
}


// 请求的二次封装
export function postJson(url,json) {
    return admin.post(url, json, {
        headers:{
            "Content-Type": "application/json; charset=utf-8"
        }
    })
}

export function get(url, params) {
    return admin.get(url, {
        params
    })
}

export {admin}