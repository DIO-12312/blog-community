import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { userApi } from '@/api'

export const useUserStore = defineStore('user', () => {
  // 状态
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref<any>(null)

  // 计算属性
  const isLoggedIn = computed(() => !!token.value)

  // 登录
  async function login(username: string, password: string) {
    const res: any = await userApi.login({ username, password })
    const newToken: string = res.data.token

    // 先解析 JWT 获取用户 ID，再拉取用户信息
    const payload = JSON.parse(atob(newToken.split('.')[1]))
    const profile: any = await userApi.getProfile(payload.user_id)
    userInfo.value = profile.data

    // 确认都成功后再保存 token
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  // 注册
  async function register(username: string, email: string, password: string) {
    await userApi.register({ username, email, password })
  }

  // 注销
  function logout() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
  }

  return { token, userInfo, isLoggedIn, login, register, logout }
})
