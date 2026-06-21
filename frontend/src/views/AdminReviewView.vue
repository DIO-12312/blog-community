<template>
  <div class="admin-page">
    <h1>审稿管理</h1>

    <div class="tabs">
      <button :class="{ active: activeTab === 'pending' }" @click="activeTab = 'pending'">待审文章</button>
      <router-link to="/admin" class="nav-back">← 返回管理面板</router-link>
    </div>

    <!-- 待审文章列表 -->
    <div class="tab-content">
      <h2>待审文章列表</h2>
      <table class="data-table">
        <thead>
          <tr>
            <th>标题</th>
            <th>作者</th>
            <th>提交时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="article in articles" :key="article.id">
            <td class="col-title">{{ article.title }}</td>
            <td>{{ article.author_id }}</td>
            <td>{{ formatDate(article.updated_at) }}</td>
            <td>
              <button class="btn-review" @click="openReview(article)">审稿</button>
            </td>
          </tr>
          <tr v-if="articles.length === 0">
            <td colspan="4" class="placeholder">暂无待审文章</td>
          </tr>
        </tbody>
      </table>
      <div class="pagination" v-if="total > size">
        <button :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
        <span>{{ page }} / {{ Math.ceil(total / size) }}</span>
        <button :disabled="page >= Math.ceil(total / size)" @click="changePage(page + 1)">下一页</button>
      </div>
    </div>

    <!-- 审稿对话框 -->
    <div v-if="reviewing" class="modal-overlay" @click.self="closeReview">
      <div class="modal">
        <h2>审稿：{{ reviewingArticle?.title }}</h2>
        <div class="article-preview">
          <pre class="raw-markdown">{{ reviewingArticle?.content }}</pre>
        </div>
        <div class="review-actions">
          <label>
            <input type="radio" v-model="action" value="approved" /> 通过
          </label>
          <label>
            <input type="radio" v-model="action" value="rejected" /> 驳回
          </label>
        </div>
        <div class="review-comment-input">
          <label>意见（可选）</label>
          <textarea v-model="comment" rows="3" placeholder="审稿意见..."></textarea>
        </div>
        <div class="modal-buttons">
          <button class="btn-cancel" @click="closeReview">取消</button>
          <button class="btn-confirm" :disabled="submitting" @click="handleReview">
            {{ submitting ? '提交中...' : '提交审稿结果' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi } from '@/api'

const activeTab = ref('pending')
const articles = ref<any[]>([])
const page = ref(1)
const size = 20
const total = ref(0)

// 审稿对话框
const reviewing = ref(false)
const reviewingArticle = ref<any>(null)
const action = ref('approved')
const comment = ref('')
const submitting = ref(false)

async function fetchPending() {
  try {
    const res: any = await adminApi.getPendingReviews(page.value, size)
    articles.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch {
    alert('获取待审文章失败')
  }
}

function changePage(p: number) {
  page.value = p
  fetchPending()
}

function openReview(article: any) {
  reviewingArticle.value = article
  action.value = 'approved'
  comment.value = ''
  reviewing.value = true
}

function closeReview() {
  reviewing.value = false
  reviewingArticle.value = null
}

async function handleReview() {
  submitting.value = true
  try {
    await adminApi.reviewArticle(reviewingArticle.value.id, {
      action: action.value,
      comment: comment.value || undefined,
    })
    closeReview()
    fetchPending()
  } catch (e: any) {
    alert(e?.message || '审稿失败')
  } finally {
    submitting.value = false
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchPending()
})
</script>

<style scoped>
.admin-page {
  max-width: 1100px;
  margin: 0 auto;
  padding: 24px;
}

h1 { font-size: 24px; margin-bottom: 20px; }

.tabs {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 2px solid #e8e8e8;
  margin-bottom: 24px;
}

.tabs button {
  padding: 10px 24px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
}

.tabs button.active {
  color: #3498db;
  border-bottom-color: #3498db;
}

.nav-back {
  font-size: 14px;
  color: #3498db;
  text-decoration: none;
}

.tab-content { min-height: 300px; }

h2 { font-size: 18px; margin-bottom: 16px; }

.data-table { width: 100%; border-collapse: collapse; }

.data-table th, .data-table td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid #eee;
  font-size: 14px;
}

.data-table th { background: #f9f9f9; font-weight: 600; color: #333; }

.col-title { max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.btn-review {
  padding: 4px 14px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.btn-review:hover { background: #2980b9; }

.placeholder { color: #999; font-size: 15px; padding: 40px 0; text-align: center; }

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-top: 20px;
}

.pagination button {
  padding: 6px 14px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
}

.pagination button:disabled { opacity: 0.4; cursor: not-allowed; }

.pagination span { font-size: 14px; color: #666; }

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 200;
}

.modal {
  background: #fff;
  border-radius: 12px;
  padding: 32px;
  width: 720px;
  max-height: 85vh;
  overflow-y: auto;
}

.modal h2 { font-size: 20px; margin-bottom: 16px; }

.article-preview {
  max-height: 400px;
  overflow-y: auto;
  background: #f5f5f5;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 20px;
}

.raw-markdown {
  font-family: 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

.review-actions {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
}

.review-actions label {
  font-size: 15px;
  cursor: pointer;
}

.review-actions input { margin-right: 6px; }

.review-comment-input { margin-bottom: 20px; }

.review-comment-input label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 6px;
}

.review-comment-input textarea {
  width: 100%;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.modal-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-cancel {
  padding: 8px 20px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-confirm {
  padding: 8px 20px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-confirm:hover { background: #2980b9; }
.btn-confirm:disabled { opacity: 0.6; }
</style>
