<template>
  <div class="home">
    <div class="search-bar">
      <input v-model="searchQuery" placeholder="搜索文章..." @keyup.enter="handleSearch" />
    </div>

    <div class="article-list">
      <ArticleCard
        v-for="article in articles"
        :key="article.id"
        :article="article"
        @click="goToArticle(article.id)"
      />
    </div>

    <div v-if="loading" class="loading">加载中...</div>
    <div v-if="!loading && articles.length === 0" class="empty">暂无文章</div>

    <!-- 分页 -->
    <div class="pagination" v-if="total > size">
      <button :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
      <span>{{ page }} / {{ Math.ceil(total / size) }}</span>
      <button :disabled="page >= Math.ceil(total / size)" @click="changePage(page + 1)">下一页</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { articleApi, searchApi } from '@/api'
import ArticleCard from '@/components/ArticleCard.vue'

const router = useRouter()

const articles = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(10)
const loading = ref(false)
const searchQuery = ref('')

async function fetchArticles() {
  loading.value = true
  try {
    const res: any = await articleApi.getList(page.value, size.value)
    articles.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

async function handleSearch() {
  if (!searchQuery.value.trim()) {
    fetchArticles()
    return
  }
  loading.value = true
  try {
    const res: any = await searchApi.search(searchQuery.value, 1, size.value)
    articles.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

function changePage(newPage: number) {
  page.value = newPage
  fetchArticles()
}

function goToArticle(id: string) {
  router.push(`/article/${id}`)
}

onMounted(fetchArticles)
</script>

<style scoped>
.home {
  max-width: 720px;
  margin: 24px auto;
  padding: 0 16px;
}

.search-bar {
  margin-bottom: 24px;
}

.search-bar input {
  width: 100%;
  padding: 10px 16px;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 15px;
  outline: none;
}

.search-bar input:focus {
  border-color: #3498db;
  box-shadow: 0 0 0 2px rgba(52, 152, 219, 0.15);
}

.loading,
.empty {
  text-align: center;
  padding: 48px 0;
  color: #999;
  font-size: 15px;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding: 24px 0;
}

.pagination button {
  padding: 6px 16px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 14px;
}

.pagination button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pagination span {
  font-size: 14px;
  color: #666;
}
</style>
