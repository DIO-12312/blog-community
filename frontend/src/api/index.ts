import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

// 请求拦截器：自动带上 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：统一处理错误
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error.response?.data || error)
  }
)

// ===== 用户 API =====
export const userApi = {
  register(data: { username: string; email: string; password: string }) {
    return api.post('/users/register', data)
  },
  login(data: { username: string; password: string }) {
    return api.post('/users/login', data)
  },
  getProfile(id: string) {
    return api.get(`/users/${id}`)
  },
}

// ===== 文章 API =====
export const articleApi = {
  getList(page = 1, size = 10) {
    return api.get('/articles', { params: { page, size } })
  },
  getDetail(id: string) {
    return api.get(`/articles/${id}`)
  },
  create(data: { title: string; content: string; category_id: string }) {
    return api.post('/articles', data)
  },
  update(id: string, data: { title: string; content: string }) {
    return api.put(`/articles/${id}`, data)
  },
  delete(id: string) {
    return api.delete(`/articles/${id}`)
  },
}

// ===== 评论 API =====
export const commentApi = {
  getByArticle(articleId: string, page = 1, size = 10) {
    return api.get(`/articles/${articleId}/comments`, { params: { page, size } })
  },
  create(articleId: string, data: { content: string; parent_id?: string }) {
    return api.post(`/articles/${articleId}/comments`, data)
  },
  delete(id: string) {
    return api.delete(`/comments/${id}`)
  },
}

// ===== 互动 API =====
export const interactionApi = {
  like(targetId: string, targetType: string) {
    return api.post('/likes', { target_id: targetId, target_type: targetType })
  },
  unlike(targetId: string, targetType: string) {
    return api.delete('/likes', { data: { target_id: targetId, target_type: targetType } })
  },
  getLikeStatus(targetId: string, targetType: string) {
    return api.get('/likes/status', { params: { target_id: targetId, target_type: targetType } })
  },
  collect(articleId: string) {
    return api.post('/collections', { article_id: articleId })
  },
  uncollect(articleId: string) {
    return api.delete(`/collections/${articleId}`)
  },
  getCollectStatus(articleId: string) {
    return api.get('/collections/status', { params: { article_id: articleId } })
  },
}

// ===== 搜索 API =====
export const searchApi = {
  search(q: string, page = 1, size = 10) {
    return api.get('/search', { params: { q, page, size } })
  },
}

// ===== 通知 API =====
export const notificationApi = {
  getList(page = 1, size = 10) {
    return api.get('/notifications', { params: { page, size } })
  },
  markAsRead(id: string) {
    return api.put(`/notifications/${id}/read`)
  },
  markAllAsRead() {
    return api.put('/notifications/read-all')
  },
  getUnreadCount() {
    return api.get('/notifications/unread-count')
  },
}

export default api
