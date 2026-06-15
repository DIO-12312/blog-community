<template>
  <div class="comment-section">
    <h3>评论 ({{ total }})</h3>

    <!-- 发表评论 -->
    <div class="comment-form" v-if="userStore.isLoggedIn">
      <textarea
        v-model="newComment"
        placeholder="写下你的评论..."
        rows="3"
      ></textarea>
      <button @click="submitComment" :disabled="submitting || !newComment.trim()">
        {{ submitting ? '提交中...' : '发表评论' }}
      </button>
    </div>
    <p v-else class="login-hint">
      <router-link to="/login">登录</router-link> 后参与评论
    </p>

    <!-- 评论列表 -->
    <div v-if="loading" class="loading">加载评论中...</div>
    <div v-else-if="comments.length === 0" class="empty">暂无评论</div>
    <div v-else class="comment-list">
      <div v-for="comment in comments" :key="comment.id" class="comment-item">
        <div class="comment-header">
          <span class="comment-author">{{ comment.username }}</span>
          <span class="comment-time">{{ comment.created_at }}</span>
        </div>
        <div class="comment-content">{{ comment.content }}</div>
        <div class="comment-actions">
          <button
            v-if="userStore.isLoggedIn"
            class="btn-reply"
            @click="toggleReply(comment.id)"
          >
            回复
          </button>
          <button
            v-if="userStore.isAdmin"
            class="btn-delete"
            @click="handleDeleteComment(comment.id)"
          >
            删除
          </button>
        </div>

        <!-- 回复输入框 -->
        <div v-if="replyTarget === comment.id" class="reply-form">
          <textarea
            v-model="replyContent"
            placeholder="写下你的回复..."
            rows="2"
          ></textarea>
          <button
            @click="submitReply(comment.id)"
            :disabled="!replyContent.trim()"
          >
            回复
          </button>
          <button class="btn-cancel" @click="cancelReply">取消</button>
        </div>

        <!-- 子评论 -->
        <div v-if="comment.children && comment.children.length" class="children">
          <div v-for="child in comment.children" :key="child.id" class="comment-item child">
            <div class="comment-header">
              <span class="comment-author">{{ child.username }}</span>
              <span class="comment-time">{{ child.created_at }}</span>
            </div>
            <div class="comment-content">{{ child.content }}</div>
            <button
              v-if="userStore.isAdmin"
              class="btn-delete"
              @click="handleDeleteComment(child.id)"
            >
              删除
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { commentApi, adminApi } from '@/api'

const props = defineProps<{
  articleId: string
}>()

const userStore = useUserStore()

const comments = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const newComment = ref('')
const submitting = ref(false)
const replyTarget = ref('')
const replyContent = ref('')

async function fetchComments() {
  loading.value = true
  try {
    const res: any = await commentApi.getByArticle(props.articleId)
    comments.value = res.data || []
    total.value = res.pagination?.total || 0
  } finally {
    loading.value = false
  }
}

async function submitComment() {
  if (!newComment.value.trim()) return
  submitting.value = true
  try {
    await commentApi.create(props.articleId, { content: newComment.value.trim() })
    newComment.value = ''
    fetchComments()
  } finally {
    submitting.value = false
  }
}

function toggleReply(commentId: string) {
  replyTarget.value = replyTarget.value === commentId ? '' : commentId
  replyContent.value = ''
}

function cancelReply() {
  replyTarget.value = ''
  replyContent.value = ''
}

async function submitReply(parentId: string) {
  if (!replyContent.value.trim()) return
  try {
    await commentApi.create(props.articleId, {
      content: replyContent.value.trim(),
      parent_id: parentId,
    })
    replyContent.value = ''
    replyTarget.value = ''
    fetchComments()
  } catch {
    // 错误由拦截器处理
  }
}

async function handleDeleteComment(commentId: string) {
  if (!confirm('确定要删除这条评论吗？')) return
  try {
    await adminApi.deleteComment(commentId)
    fetchComments()
  } catch {
    // 错误由拦截器处理
  }
}

onMounted(fetchComments)
</script>

<style scoped>
.comment-section {
  margin-top: 32px;
}

.comment-section h3 {
  font-size: 18px;
  margin-bottom: 16px;
}

.comment-form {
  margin-bottom: 24px;
}

.comment-form textarea,
.reply-form textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.comment-form button,
.reply-form button {
  margin-top: 8px;
  padding: 6px 16px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.reply-form .btn-cancel {
  background: #eee;
  color: #666;
  margin-left: 8px;
}

.login-hint {
  color: #999;
  margin-bottom: 24px;
}

.login-hint a {
  color: #3498db;
}

.loading,
.empty {
  text-align: center;
  padding: 24px 0;
  color: #999;
}

.comment-item {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.comment-header {
  margin-bottom: 6px;
}

.comment-author {
  font-weight: 600;
  font-size: 14px;
  color: #333;
}

.comment-time {
  font-size: 12px;
  color: #999;
  margin-left: 12px;
}

.comment-content {
  font-size: 14px;
  line-height: 1.6;
  color: #444;
}

.comment-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}

.btn-reply {
  padding: 2px 8px;
  background: none;
  border: none;
  color: #999;
  font-size: 13px;
  cursor: pointer;
}

.btn-reply:hover {
  color: #3498db;
}

.btn-delete {
  padding: 2px 8px;
  background: none;
  border: none;
  color: #e74c3c;
  font-size: 12px;
  cursor: pointer;
}

.btn-delete:hover {
  text-decoration: underline;
}

.reply-form {
  margin-top: 8px;
}

.children {
  margin-left: 24px;
  padding-left: 16px;
  border-left: 2px solid #f0f0f0;
}
</style>
