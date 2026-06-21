<template>
  <div class="editor-container">
    <h1>{{ isEdit ? '编辑文章' : '写文章' }}</h1>
    <form @submit.prevent>
      <div class="form-group">
        <label>标题</label>
        <input
          v-model="title"
          type="text"
          placeholder="请输入文章标题"
          required
        />
      </div>
      <div class="form-group">
        <label>分类 ID</label>
        <input
          v-model="categoryId"
          type="text"
          placeholder="请输入分类 ID"
        />
      </div>
      <div class="form-group">
        <label>内容 (Markdown)</label>
        <textarea
          v-model="content"
          rows="16"
          placeholder="支持 Markdown 格式..."
          required
        ></textarea>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="form-actions">
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

      <!-- 重新审核按钮（驳回后重投） -->
      <button
        v-if="reviewStatus === 'draft' && reviewHistory.length > 0"
        type="button"
        class="btn-submit-review"
        :disabled="submittingReview"
        @click="handleResubmitReview"
      >
        {{ submittingReview ? '提交中...' : '重新审核' }}
      </button>

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
const categoryId = ref('')
const error = ref('')
const submitting = ref(false)

const reviewStatus = ref('')
const reviewHistory = ref<any[]>([])
const submittingReview = ref(false)

const editId = (route.params.id as string) || ''

// 加载审稿信息
async function fetchReviewInfo() {
  try {
    const res: any = await articleApi.getReviewHistory(editId)
    reviewHistory.value = res.data || []
  } catch { /* 忽略 */ }
}

// 重新审核（驳回后重投，仅调用审稿 API）
async function handleResubmitReview() {
  if (!confirm('确认重新提交审核？提交后将无法编辑。')) return
  submittingReview.value = true
  try {
    await articleApi.submitReview(editId)
    reviewStatus.value = 'pending_review'
  } catch (e: any) {
    error.value = e?.message || '重新审核失败'
  } finally {
    submittingReview.value = false
  }
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
      title.value = res.data.title
      content.value = res.data.content
      categoryId.value = res.data.category_id || ''
      reviewStatus.value = res.data.status
      await fetchReviewInfo()
    } catch {
      error.value = '加载文章失败'
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
      })
    } else {
      const res: any = await articleApi.create({
        title: title.value,
        content: content.value,
        category_id: categoryId.value,
      })
      router.push(`/editor/${res.data.id}`)
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
      })
      await articleApi.submitReview(editId)
      reviewStatus.value = 'pending_review'
    } else {
      const res: any = await articleApi.create({
        title: title.value,
        content: content.value,
        category_id: categoryId.value,
      })
      await articleApi.submitReview(res.data.id)
      router.push(`/editor/${res.data.id}`)
    }
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
