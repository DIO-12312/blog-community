<template>
  <div class="notification-container">
    <div class="header">
      <h1>通知</h1>
      <button
        v-if="notifications.length > 0"
        class="btn-read-all"
        @click="handleMarkAllRead"
      >
        全部已读
      </button>
    </div>

    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="notifications.length === 0" class="empty">暂无通知</div>
    <div v-else class="notification-list">
      <div
        v-for="item in notifications"
        :key="item.id"
        class="notification-item"
        :class="{ unread: !item.is_read }"
        @click="handleClick(item)"
      >
        <div class="notification-content">
          <p>{{ item.content }}</p>
          <span class="time">{{ formatDate(item.created_at) }}</span>
        </div>
        <button
          v-if="!item.is_read"
          class="btn-mark-read"
          @click.stop="handleMarkRead(item.id)"
        >
          标为已读
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { notificationApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()

const notifications = ref<any[]>([])
const loading = ref(false)
const error = ref('')

async function fetchNotifications() {
  loading.value = true
  error.value = ''
  try {
    const res: any = await notificationApi.getList()
    notifications.value = res.data || []
  } catch (e: any) {
    // 401 已由拦截器统一处理（清空登录状态 + 跳转）
    if (e?.code !== 401) {
      error.value = '加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

async function handleMarkRead(id: string) {
  try {
    await notificationApi.markAsRead(id)
    const target = notifications.value.find((n) => n.id === id)
    if (target) {
      target.is_read = true
      userStore.unreadCount--
    }
  } catch {
    // 错误由拦截器处理
  }
}

async function handleMarkAllRead() {
  try {
    await notificationApi.markAllAsRead()
    notifications.value.forEach((n) => (n.is_read = true))
    userStore.unreadCount = 0
  } catch {
    // 错误由拦截器处理
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN')
}

function handleClick(item: any) {
  if (!item.is_read) {
    handleMarkRead(item.id)
  }

  const sourceId = item.source_id
  if (!sourceId) return

  switch (item.type) {
    case 'new_submission':
      router.push('/admin/reviews')
      break
    case 'review_rejected':
      router.push(`/editor/${sourceId}`)
      break
    default:
      // comment, follow, like → navigate to article
      router.push(`/article/${sourceId}`)
  }
}

onMounted(fetchNotifications)
</script>

<style scoped>
.notification-container {
  max-width: 680px;
  margin: 32px auto;
  padding: 24px;
  background: #fff;
  border-radius: 8px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h1 {
  font-size: 22px;
}

.btn-read-all {
  padding: 4px 12px;
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: #666;
}

.btn-read-all:hover {
  border-color: #3498db;
  color: #3498db;
}

.loading,
.empty {
  text-align: center;
  padding: 48px 0;
  color: #999;
}

.notification-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.15s;
}

.notification-item:hover {
  background: #fafafa;
}

.notification-item.unread {
  background: #f0f7ff;
}

.notification-content p {
  font-size: 15px;
  color: #333;
  margin-bottom: 4px;
}

.time {
  font-size: 12px;
  color: #999;
}

.btn-mark-read {
  padding: 4px 10px;
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  color: #999;
  flex-shrink: 0;
  margin-left: 12px;
}

.btn-mark-read:hover {
  border-color: #3498db;
  color: #3498db;
}
</style>
