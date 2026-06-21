<template>
  <div class="profile">
    <!-- 用户信息卡片 -->
    <div class="user-card">
      <div class="user-avatar">
        {{ (userStore.userInfo?.username || 'U')[0].toUpperCase() }}
      </div>
      <div class="user-info">
        <h2 class="user-name">{{ userStore.userInfo?.username }}</h2>
        <p class="user-email">{{ userStore.userInfo?.email }}</p>
        <p class="user-joined">{{ formatDate(userStore.userInfo?.created_at) }} 加入</p>
      </div>
    </div>

    <!-- 标签页 -->
    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tab', { active: currentTab === tab.key }]"
        @click="currentTab = tab.key"
      >
        {{ tab.label }} ({{ tab.count }})
      </button>
    </div>

    <!-- 文章列表 -->
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="currentArticles.length === 0" class="empty">暂无文章</div>
    <div v-else class="article-list">
      <div v-for="article in currentArticles" :key="article.id" class="article-item">
        <div class="item-main" @click="goToArticle(article)">
          <h3 class="item-title">{{ article.title }}</h3>
          <p class="item-summary">{{ article.summary || stripMarkdown(article.content) }}</p>
          <div class="item-meta">
            <span>{{ formatDate(article.created_at) }}</span>
            <span>{{ article.view_count || 0 }} 阅读</span>
            <span>{{ article.like_count || 0 }} 赞</span>
            <span>{{ article.comment_count || 0 }} 评论</span>
            <span v-if="article.category" class="item-category">{{ article.category }}</span>
          </div>
        </div>
        <div class="item-actions">
          <button
            v-if="article.status === 'draft'"
            class="btn-edit"
            @click.stop="goToEditor(article.id)"
          >编辑</button>
          <button
            v-if="article.status === 'published'"
            class="btn-view"
            @click.stop="goToArticle(article)"
          >查看</button>
          <button
            v-if="article.status === 'pending_review'"
            class="btn-pending"
            disabled
          >审核中</button>
          <button
            v-if="article.status === 'draft'"
            class="btn-delete"
            @click.stop="handleDelete(article.id)"
          >删除</button>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="pagination" v-if="total > pageSize">
      <button :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
      <span>{{ page }} / {{ Math.ceil(total / pageSize) }}</span>
      <button :disabled="page >= Math.ceil(total / pageSize)" @click="changePage(page + 1)">下一页</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { articleApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()

const articles = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const loading = ref(false)
const currentTab = ref('published')

const tabs = computed(() => [
  { key: 'published', label: '已发布', count: publishedArticles.value.length },
  { key: 'pending_review', label: '待审核', count: pendingArticles.value.length },
  { key: 'draft', label: '草稿', count: drafts.value.length },
])

const drafts = computed(() => articles.value.filter(a => a.status === 'draft'))
const pendingArticles = computed(() => articles.value.filter(a => a.status === 'pending_review'))
const publishedArticles = computed(() => articles.value.filter(a => a.status === 'published'))

const currentArticles = computed(() => {
  switch (currentTab.value) {
    case 'draft': return drafts.value
    case 'pending_review': return pendingArticles.value
    default: return publishedArticles.value
  }
})

function stripMarkdown(text: string) {
  if (!text) return ''
  const plain = text.replace(/[#*`>\[\]]/g, '').replace(/\n/g, ' ')
  return plain.length > 200 ? plain.slice(0, 200) + '...' : plain
}

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

function goToArticle(article: any) {
  router.push(`/article/${article.id}`)
}

function goToEditor(id: string) {
  router.push(`/editor/${id}`)
}

async function handleDelete(id: string) {
  if (!confirm('确定要删除这篇文章吗？')) return
  try {
    await articleApi.delete(id)
    articles.value = articles.value.filter(a => a.id !== id)
  } catch {
    alert('删除失败')
  }
}

async function fetchArticles() {
  loading.value = true
  try {
    const res: any = await articleApi.getMyArticles(page.value, pageSize.value)
    articles.value = res.data || []
    total.value = res.pagination?.total || 0
  } finally {
    loading.value = false
  }
}

function changePage(p: number) {
  page.value = p
  fetchArticles()
}

onMounted(() => {
  fetchArticles()
})
</script>

<style scoped>
.profile {
  max-width: 800px;
  margin: 24px auto;
  padding: 0 16px;
}

.user-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px;
  background: #fff;
  border-radius: 8px;
  margin-bottom: 24px;
}

.user-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: #3498db;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  font-weight: 600;
  flex-shrink: 0;
}

.user-name {
  font-size: 20px;
  margin: 0;
}

.user-email {
  color: #666;
  font-size: 14px;
  margin: 4px 0;
}

.user-joined {
  color: #999;
  font-size: 12px;
  margin: 0;
}

.tabs {
  display: flex;
  gap: 0;
  border-bottom: 2px solid #e8e8e8;
  margin-bottom: 16px;
}

.tab {
  padding: 8px 20px;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: color 0.2s, border-color 0.2s;
}

.tab:hover {
  color: #3498db;
}

.tab.active {
  color: #3498db;
  border-bottom-color: #3498db;
}

.article-item {
  background: #fff;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.item-main {
  flex: 1;
  cursor: pointer;
  min-width: 0;
}

.item-title {
  font-size: 16px;
  margin: 0 0 6px;
  color: #222;
}

.item-summary {
  font-size: 13px;
  color: #888;
  margin: 0 0 8px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.item-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #aaa;
  flex-wrap: wrap;
  align-items: center;
}

.item-category {
  background: #e8f5e9;
  color: #2e7d32;
  padding: 1px 8px;
  border-radius: 3px;
  font-size: 11px;
}

.item-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.btn-edit,
.btn-view {
  padding: 4px 12px;
  border: 1px solid #3498db;
  background: none;
  color: #3498db;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
}

.btn-edit:hover,
.btn-view:hover {
  background: #3498db;
  color: #fff;
}

.btn-pending {
  padding: 4px 12px;
  border: 1px solid #f0ad4e;
  background: none;
  color: #f0ad4e;
  border-radius: 4px;
  font-size: 13px;
  white-space: nowrap;
}

.btn-delete {
  padding: 4px 12px;
  border: 1px solid #e74c3c;
  background: none;
  color: #e74c3c;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
}

.btn-delete:hover {
  background: #e74c3c;
  color: #fff;
}

.loading,
.empty {
  text-align: center;
  padding: 48px;
  color: #999;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 24px;
}

.pagination button {
  padding: 6px 16px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.pagination button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pagination span {
  font-size: 13px;
  color: #666;
}
</style>
