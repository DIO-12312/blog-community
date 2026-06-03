<template>
  <div class="article-card" @click="$emit('click')">
    <h2 class="card-title">{{ article.title }}</h2>
    <p class="card-summary">{{ summary }}</p>
    <div class="card-meta">
      <span>{{ article.username }}</span>
      <span>{{ article.created_at }}</span>
      <span>{{ article.view_count || 0 }} 阅读</span>
      <span>{{ article.comment_count || 0 }} 评论</span>
      <span>{{ article.like_count || 0 }} 赞</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  article: {
    id: string
    title: string
    content: string
    username?: string
    created_at?: string
    view_count?: number
    comment_count?: number
    like_count?: number
  }
}>()

defineEmits<{
  click: []
}>()

const summary = computed(() => {
  const text = props.article.content || ''
  // 简单去除 markdown 标记，取前 200 字符
  const plain = text.replace(/[#*`>\[\]]/g, '').replace(/\n/g, ' ')
  return plain.length > 200 ? plain.slice(0, 200) + '...' : plain
})
</script>

<style scoped>
.article-card {
  background: #fff;
  padding: 20px 24px;
  border-radius: 8px;
  margin-bottom: 16px;
  cursor: pointer;
  transition: box-shadow 0.2s;
}
.article-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}
.card-title {
  font-size: 18px;
  margin-bottom: 8px;
  color: #222;
}
.card-summary {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
  line-height: 1.6;
}
.card-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #999;
}
</style>
