<template>
  <div class="editor-container">
    <h1>{{ isEdit ? '编辑文章' : '写文章' }}</h1>
    <form @submit.prevent="handleSubmit">
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
      <button type="submit" :disabled="submitting">
        {{ submitting ? '保存中...' : '发布' }}
      </button>
    </form>
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

const editId = (route.params.id as string) || ''

onMounted(async () => {
  if (editId) {
    isEdit.value = true
    try {
      const res: any = await articleApi.getDetail(editId)
      title.value = res.data.title
      content.value = res.data.content
    } catch {
      error.value = '加载文章失败'
    }
  }
})

async function handleSubmit() {
  submitting.value = true
  error.value = ''
  try {
    if (isEdit.value) {
      await articleApi.update(editId, {
        title: title.value,
        content: content.value,
      })
    } else {
      await articleApi.create({
        title: title.value,
        content: content.value,
        category_id: categoryId.value,
      })
    }
    router.push('/')
  } catch (e: any) {
    error.value = e.message || '保存失败'
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

button[type='submit'] {
  padding: 10px 32px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
}

button[type='submit']:disabled {
  opacity: 0.6;
}
</style>
