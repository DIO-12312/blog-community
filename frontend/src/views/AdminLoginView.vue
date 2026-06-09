<template>
  <div class="admin-login-page">
    <div class="login-card">
      <div class="card-header">
        <h1>管理员登录</h1>
        <p>仅限管理员访问</p>
      </div>
      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="username" type="text" placeholder="请输入管理员账号" required autocomplete="username" />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="password" type="password" placeholder="请输入密码" required autocomplete="current-password" />
        </div>
        <p v-if="error" class="error">{{ error }}</p>
        <button type="submit" :disabled="loading">
          {{ loading ? '登录中...' : '登 录' }}
        </button>
      </form>
      <p class="back-link">
        <router-link to="/login">← 返回普通用户登录</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  loading.value = true
  error.value = ''
  try {
    await userStore.login(username.value, password.value)
    if (!userStore.isAdmin) {
      userStore.logout()
      error.value = '该账号无管理员权限'
      return
    }
    router.push('/admin')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.admin-login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1a1a2e;
}

.login-card {
  width: 400px;
  padding: 40px 36px;
  background: #16213e;
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.card-header {
  text-align: center;
  margin-bottom: 32px;
}

.card-header h1 {
  font-size: 22px;
  color: #e0e0e0;
  margin: 0 0 8px;
}

.card-header p {
  font-size: 13px;
  color: #888;
  margin: 0;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #aaa;
}

.form-group input {
  width: 100%;
  padding: 10px 14px;
  background: #0f3460;
  border: 1px solid #1a1a4e;
  border-radius: 4px;
  font-size: 14px;
  color: #e0e0e0;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-group input::placeholder {
  color: #567;
}

.form-group input:focus {
  border-color: #e94560;
}

.error {
  color: #e94560;
  font-size: 13px;
  margin-bottom: 12px;
  text-align: center;
}

button {
  width: 100%;
  padding: 12px;
  background: #e94560;
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

button:hover {
  background: #d63851;
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.back-link {
  text-align: center;
  margin-top: 20px;
}

.back-link a {
  color: #888;
  font-size: 13px;
  text-decoration: none;
}

.back-link a:hover {
  color: #aaa;
}
</style>
