<template>
  <nav class="navbar">
    <div class="nav-left">
      <router-link to="/" class="logo">博客社区</router-link>
      <router-link to="/" class="nav-link">首页</router-link>
    </div>
    <div class="nav-right">
      <template v-if="userStore.isLoggedIn">
        <router-link to="/editor" class="nav-link">写文章</router-link>
        <router-link to="/notifications" class="nav-link">
          通知
          <span v-if="unreadCount > 0" class="badge">{{ unreadCount }}</span>
        </router-link>
        <span class="nav-user">{{ userStore.userInfo?.username }}</span>
        <button @click="handleLogout" class="btn-logout">退出</button>
      </template>
      <template v-else>
        <router-link to="/login" class="nav-link">登录</router-link>
        <router-link to="/register" class="nav-link">注册</router-link>
      </template>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { notificationApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()
const unreadCount = ref(0)

async function fetchUnreadCount() {
  try {
    const res: any = await notificationApi.getUnreadCount()
    unreadCount.value = res.data.count
  } catch {
    // 静默失败
  }
}

function handleLogout() {
  userStore.logout()
  router.push('/')
}

onMounted(() => {
  if (userStore.isLoggedIn) {
    fetchUnreadCount()
  }
})
</script>

<style scoped>
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  position: sticky;
  top: 0;
  z-index: 100;
}

.nav-left,
.nav-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.logo {
  font-size: 18px;
  font-weight: 700;
  color: #3498db;
  text-decoration: none;
  margin-right: 16px;
}

.nav-link {
  color: #555;
  text-decoration: none;
  font-size: 14px;
  position: relative;
}

.nav-link:hover {
  color: #3498db;
}

.nav-user {
  font-size: 14px;
  color: #333;
}

.badge {
  background: #e74c3c;
  color: #fff;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 10px;
  margin-left: 4px;
}

.btn-logout {
  padding: 4px 12px;
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: #666;
}

.btn-logout:hover {
  border-color: #e74c3c;
  color: #e74c3c;
}
</style>
