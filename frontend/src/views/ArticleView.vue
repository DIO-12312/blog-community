<template>
  <div class="article-detail" v-if="article">
    <h1>{{ article.title }}</h1>
    <div class="meta">
      <span>{{ article.username }}</span>
      <span>{{ article.created_at }}</span>
      <span>{{ article.view_count }} 阅读</span>
    </div>

    <div class="content" v-html="renderedContent"></div>

    <!-- 管理员操作 -->
    <div v-if="userStore.isAdmin" class="admin-actions">
      <button class="btn-delete" @click="handleAdminDelete">删除文章</button>
    </div>

    <!-- 互动按钮 -->
    <div class="actions">
      <button @click="toggleLike" :class="{ active: isLiked }">
        {{ isLiked ? '已赞' : '点赞' }} ({{ likeCount }})
      </button>
      <button @click="toggleCollect" :class="{ active: isCollected }">
        {{ isCollected ? '已收藏' : '收藏' }} ({{ collectCount }})
      </button>
    </div>

    <!-- 评论区 -->
    <CommentList :article-id="articleId" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { articleApi, interactionApi, adminApi } from '@/api'
import CommentList from '@/components/CommentList.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const articleId = route.params.id as string

const article = ref<any>(null)
const isLiked = ref(false)
const likeCount = ref(0)
const isCollected = ref(false)
const collectCount = ref(0)

// 简单的 Markdown 渲染
const renderedContent = computed(() => {
  if (!article.value) return ''
  return article.value.content
    .replace(/\n/g, '<br>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/`(.*?)`/g, '<code>$1</code>')
})

async function fetchArticle() {
  const res: any = await articleApi.getDetail(articleId)
  article.value = res.data
}

async function fetchInteractionStatus() {
  const [likeRes, collectRes]: any[] = await Promise.all([
    interactionApi.getLikeStatus(articleId, 'article'),
    interactionApi.getCollectStatus(articleId),
  ])
  isLiked.value = likeRes.data.is_liked
  likeCount.value = likeRes.data.count
  isCollected.value = collectRes.data.is_collected
  collectCount.value = collectRes.data.count
}

async function toggleLike() {
  if (isLiked.value) {
    await interactionApi.unlike(articleId, 'article')
    isLiked.value = false
    likeCount.value--
  } else {
    await interactionApi.like(articleId, 'article')
    isLiked.value = true
    likeCount.value++
  }
}

async function toggleCollect() {
  if (isCollected.value) {
    await interactionApi.uncollect(articleId)
    isCollected.value = false
    collectCount.value--
  } else {
    await interactionApi.collect(articleId)
    isCollected.value = true
    collectCount.value++
  }
}

async function handleAdminDelete() {
  if (!confirm('确定要删除这篇文章吗？')) return
  try {
    await adminApi.deleteArticle(articleId)
    router.push('/')
  } catch {
    // 错误由拦截器处理
  }
}

onMounted(() => {
  fetchArticle()
  fetchInteractionStatus()
})
</script>

<style scoped>
.article-detail {
  max-width: 760px;
  margin: 32px auto;
  padding: 32px;
  background: #fff;
  border-radius: 8px;
}

.article-detail h1 {
  font-size: 28px;
  margin-bottom: 16px;
  color: #222;
}

.meta {
  display: flex;
  gap: 16px;
  font-size: 14px;
  color: #999;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.content {
  font-size: 16px;
  line-height: 1.8;
  color: #333;
  margin-bottom: 32px;
}

.admin-actions {
  padding: 12px 0;
  border-top: 1px solid #f0f0f0;
}

.btn-delete {
  padding: 6px 16px;
  background: none;
  border: 1px solid #e74c3c;
  border-radius: 4px;
  color: #e74c3c;
  cursor: pointer;
  font-size: 13px;
}

.btn-delete:hover {
  background: #e74c3c;
  color: #fff;
}

.actions {
  display: flex;
  gap: 12px;
  padding: 16px 0;
  border-top: 1px solid #f0f0f0;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 24px;
}

.actions button {
  padding: 8px 20px;
  border: 1px solid #ddd;
  border-radius: 6px;
  background: #fff;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  transition: all 0.2s;
}

.actions button:hover {
  border-color: #3498db;
  color: #3498db;
}

.actions button.active {
  background: #3498db;
  color: #fff;
  border-color: #3498db;
}
</style>
