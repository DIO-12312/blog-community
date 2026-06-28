<template>
  <div class="editor-container">
    <h1>{{ isEdit ? '编辑文章' : '写文章' }}</h1>
    <div v-if="hasPendingReview && !isEdit" class="pending-banner">存在正在审核的文章，审核完成前无法编写新文章。</div>
    <form @submit.prevent>
      <div class="form-group">
        <label>标题</label>
        <input
          v-model="title"
          type="text"
          placeholder="请输入文章标题"
          required
          :disabled="hasPendingReview && !isEdit"
        />
      </div>
      <div class="form-group">
        <label>分类</label>
        <input
          v-model="category"
          type="text"
          placeholder="请输入分类"
          :disabled="hasPendingReview && !isEdit"
        />
      </div>
      <div class="form-group">
        <label>内容 (Markdown)</label>
        <textarea
          v-model="content"
          rows="16"
          placeholder="支持 Markdown 格式..."
          required
          :disabled="hasPendingReview && !isEdit"
        ></textarea>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div v-if="hasPendingReview && !isEdit" class="form-actions"></div>
      <div v-else-if="reviewStatus === 'pending_review'" class="form-actions">
        <button type="button" class="btn-reviewing" disabled>正在审核</button>
      </div>
      <div v-else class="form-actions">
        <button type="button" class="btn-submit-review" :disabled="submitting" @click="handleSubmitForReview">
          {{ submitting ? '提交中...' : '提交审核' }}
        </button>
        <button type="button" class="btn-save-draft" :disabled="submitting" @click="handleSaveDraft">
          保存草稿
        </button>
      </div>
    </form>

    <!-- 审稿状态（仅编辑已有文章时显示） -->
    <div v-if="isEdit && reviewStatus" class="review-section">
      <div class="review-status" :class="'status-' + reviewStatus">
        <span v-if="reviewStatus === 'pending_review'">审核中，暂不可编辑</span>
        <span v-else-if="reviewStatus === 'published'">已通过审核并发布</span>
        <span v-else-if="reviewStatus === 'draft' && reviewHistory.length > 0">已被退回，可修改后重新提交</span>
      </div>

      <!-- 审稿历史 -->
      <div v-if="reviewHistory.length > 0" class="review-history">
        <h3>审稿记录</h3>
        <div v-for="record in reviewHistory" :key="record.id" class="review-record">
          <span :class="record.action === 'approved' ? 'tag-approved' : 'tag-rejected'">
            {{ record.action === 'approved' ? '通过' : '驳回' }}
          </span>
          <span class="review-time">{{ formatDate(record.created_at) }}</span>
          <p v-if="record.comment" class="review-comment">{{ record.comment }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { articleApi } from '@/api'

const route = useRoute()
const router = useRouter()

const isEdit = ref(false)
const title = ref('')
const content = ref('')
const category = ref('')
const error = ref('')
const submitting = ref(false)

const reviewStatus = ref('')
const reviewHistory = ref<any[]>([])
const hasPendingReview = ref(false)

const editId = (route.params.id as string) || ''

// 加载审稿信息
async function fetchReviewInfo() {
  try {
    const res: any = await articleApi.getReviewHistory(editId)
    reviewHistory.value = res.data || []
  } catch { /* 忽略 */ }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(async () => {
  if (editId) {
    isEdit.value = true
    try {
      const res: any = await articleApi.getDetail(editId)
      const status = res.data.status

      // 已通过审核 → 清空编辑器，写新文章
      if (status === 'published') {
        title.value = ''
        content.value = ''
        category.value = ''
        isEdit.value = false
        reviewStatus.value = ''
        return
      }

      title.value = res.data.title
      content.value = res.data.content
      category.value = res.data.category || ''
      reviewStatus.value = status
      await fetchReviewInfo()
    } catch {
      error.value = '加载文章失败'
    }
  } else {
    // 检查是否有正在审核的文章
    try {
      const myArticles: any = await articleApi.getMyArticles(1, 10)
      hasPendingReview.value = (myArticles.data || []).some((a: any) => a.status === 'pending_review')
    } catch { /* 忽略 */ }

    // 加载用户的唯一草稿
    if (!hasPendingReview.value) {
      try {
        const res: any = await articleApi.getDraft()
        if (res.data) {
          title.value = res.data.title
          content.value = res.data.content
          category.value = res.data.category || ''
        }
      } catch { /* 无草稿，显示空白编辑器 */ }
    }
  }
})

// 保存草稿（不提交审核）
async function handleSaveDraft() {
  submitting.value = true
  error.value = ''
  try {
    if (isEdit.value) {
      await articleApi.update(editId, {
        title: title.value,
        content: content.value,
        category: category.value,
      })
    } else {
      await articleApi.saveDraft({
        title: title.value,
        content: content.value,
        category: category.value,
      })
    }
  } catch (e: any) {
    error.value = e.message || '保存失败'
  } finally {
    submitting.value = false
  }
}

// 提交审核（保存 + 提交审核）
async function handleSubmitForReview() {
  submitting.value = true
  error.value = ''
  try {
    if (isEdit.value) {
      await articleApi.update(editId, {
        title: title.value,
        content: content.value,
        category: category.value,
      })
      await articleApi.submitReview(editId)
    } else {
      const res: any = await articleApi.saveDraft({
        title: title.value,
        content: content.value,
        category: category.value,
      })
      await articleApi.submitReview(res.data.id)
    }
    router.push('/submit-success')
  } catch (e: any) {
    error.value = e.message || '提交失败'
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.editor-container {
  max-width: 760px;
  margin: 32px auto;
  padding: 32px;
  background: #fff;
  border-radius: 8px;
}

.editor-container h1 {
  margin-bottom: 24px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  font-weight: 600;
  margin-bottom: 6px;
  font-size: 14px;
}

.form-group input,
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 15px;
  font-family: inherit;
  outline: none;
}

.form-group input:focus,
.form-group textarea:focus {
  border-color: #3498db;
  box-shadow: 0 0 0 2px rgba(52, 152, 219, 0.15);
}

.error {
  color: #e74c3c;
}

.pending-banner {
  padding: 14px 18px;
  background: #fff3cd;
  color: #856404;
  border: 1px solid #ffc107;
  border-radius: 6px;
  margin-bottom: 20px;
  font-size: 14px;
}

.form-actions {
  display: flex;
  gap: 12px;
}

.btn-submit-review {
  padding: 10px 32px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
}

.btn-submit-review:hover {
  background: #2980b9;
}

.btn-submit-review:disabled {
  opacity: 0.6;
}

.btn-save-draft {
  padding: 10px 32px;
  background: #fff;
  color: #555;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
}

.btn-save-draft:hover {
  border-color: #3498db;
  color: #3498db;
}

.btn-save-draft:disabled {
  opacity: 0.6;
}

.btn-reviewing {
  padding: 10px 32px;
  background: #f0ad4e;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: not-allowed;
}

.review-section {
  margin-top: 24px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}

.review-status {
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 14px;
  margin-bottom: 16px;
}

.status-pending_review {
  background: #fff3cd;
  color: #856404;
}

.status-published {
  background: #d4edda;
  color: #155724;
}

.status-draft {
  background: #f8d7da;
  color: #721c24;
}

.btn-submit-review {
  padding: 10px 28px;
  background: #27ae60;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
  margin-bottom: 16px;
}

.btn-submit-review:hover {
  background: #219a52;
}

.btn-submit-review:disabled {
  opacity: 0.6;
}

.review-history {
  margin-top: 20px;
}

.review-history h3 {
  font-size: 16px;
  margin-bottom: 12px;
  color: #333;
}

.review-record {
  padding: 12px;
  border-bottom: 1px solid #eee;
}

.tag-approved {
  color: #27ae60;
  font-weight: 600;
}

.tag-rejected {
  color: #e74c3c;
  font-weight: 600;
}

.review-time {
  margin-left: 12px;
  font-size: 13px;
  color: #999;
}

.review-comment {
  margin-top: 6px;
  font-size: 14px;
  color: #555;
  line-height: 1.5;
}
</style>
