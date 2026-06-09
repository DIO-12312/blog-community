<template>
  <div class="admin-page">
    <h1>管理员面板</h1>

    <div class="tabs">
      <button :class="{ active: activeTab === 'users' }" @click="switchTab('users')">用户管理</button>
      <button :class="{ active: activeTab === 'articles' }" @click="switchTab('articles')">文章管理</button>
      <button :class="{ active: activeTab === 'comments' }" @click="switchTab('comments')">评论管理</button>
    </div>

    <!-- 用户管理 -->
    <div v-if="activeTab === 'users'" class="tab-content">
      <h2>用户列表</h2>
      <table class="data-table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>邮箱</th>
            <th>角色</th>
            <th>状态</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.username }}</td>
            <td>{{ user.email }}</td>
            <td>
              <span :class="user.role === 'admin' ? 'badge-admin' : 'badge-user'">
                {{ user.role === 'admin' ? '管理员' : '用户' }}
              </span>
            </td>
            <td>
              <span :class="user.banned ? 'badge-banned' : 'badge-active'">
                {{ user.banned ? '已封禁' : '正常' }}
              </span>
            </td>
            <td>{{ formatDate(user.created_at) }}</td>
            <td>
              <button
                v-if="!user.banned && user.role !== 'admin'"
                class="btn-ban"
                @click="handleBan(user.id)"
              >
                封禁
              </button>
              <button
                v-if="user.banned"
                class="btn-unban"
                @click="handleUnban(user.id)"
              >
                解封
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="pagination" v-if="userTotal > userSize">
        <button :disabled="userPage <= 1" @click="changeUserPage(userPage - 1)">上一页</button>
        <span>{{ userPage }} / {{ Math.ceil(userTotal / userSize) }}</span>
        <button :disabled="userPage >= Math.ceil(userTotal / userSize)" @click="changeUserPage(userPage + 1)">下一页</button>
      </div>
    </div>

    <!-- 文章管理 -->
    <div v-else-if="activeTab === 'articles'" class="tab-content">
      <h2>文章列表</h2>
      <table class="data-table">
        <thead>
          <tr>
            <th>标题</th>
            <th>作者</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="article in articles" :key="article.id">
            <td class="col-title">{{ article.title }}</td>
            <td>{{ article.author_id }}</td>
            <td>
              <span :class="article.deleted_at ? 'badge-deleted' : article.status === 'published' ? 'badge-active' : 'badge-draft'">
                {{ article.deleted_at ? '已删除' : article.status === 'published' ? '已发布' : '草稿' }}
              </span>
            </td>
            <td>{{ formatDate(article.created_at) }}</td>
            <td>
              <button
                class="btn-delete"
                @click="handleDeleteArticle(article.id)"
              >
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="pagination" v-if="articleTotal > articleSize">
        <button :disabled="articlePage <= 1" @click="changeArticlePage(articlePage - 1)">上一页</button>
        <span>{{ articlePage }} / {{ Math.ceil(articleTotal / articleSize) }}</span>
        <button :disabled="articlePage >= Math.ceil(articleTotal / articleSize)" @click="changeArticlePage(articlePage + 1)">下一页</button>
      </div>
    </div>

    <!-- 评论管理 -->
    <div v-else-if="activeTab === 'comments'" class="tab-content">
      <h2>评论列表</h2>
      <table class="data-table">
        <thead>
          <tr>
            <th>评论内容</th>
            <th>评论者</th>
            <th>文章ID</th>
            <th>状态</th>
            <th>时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="comment in comments" :key="comment.id">
            <td class="col-title">{{ comment.content }}</td>
            <td>{{ comment.username }}</td>
            <td>{{ comment.article_id }}</td>
            <td>
              <span :class="comment.deleted_at ? 'badge-deleted' : 'badge-active'">
                {{ comment.deleted_at ? '已删除' : '正常' }}
              </span>
            </td>
            <td>{{ formatDate(comment.created_at) }}</td>
            <td>
              <button
                v-if="!comment.deleted_at"
                class="btn-delete"
                @click="handleDeleteComment(comment.id)"
              >
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="pagination" v-if="commentTotal > commentSize">
        <button :disabled="commentPage <= 1" @click="changeCommentPage(commentPage - 1)">上一页</button>
        <span>{{ commentPage }} / {{ Math.ceil(commentTotal / commentSize) }}</span>
        <button :disabled="commentPage >= Math.ceil(commentTotal / commentSize)" @click="changeCommentPage(commentPage + 1)">下一页</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi } from '@/api'

const activeTab = ref('users')

function switchTab(tab: string) {
  activeTab.value = tab
  if (tab === 'users') fetchUsers()
  else if (tab === 'articles') fetchArticles()
  else if (tab === 'comments') fetchComments()
}

// 用户管理
const users = ref<any[]>([])
const userPage = ref(1)
const userSize = 20
const userTotal = ref(0)

async function fetchUsers() {
  try {
    const res: any = await adminApi.getUsers(userPage.value, userSize)
    users.value = res.data || []
    userTotal.value = res.pagination?.total || 0
  } catch {
    alert('获取用户列表失败')
  }
}

function changeUserPage(page: number) {
  userPage.value = page
  fetchUsers()
}

async function handleBan(id: string) {
  if (!confirm('确定要封禁该用户吗？封禁后该用户将无法登录。')) return
  try {
    await adminApi.banUser(id)
    fetchUsers()
  } catch (e: any) {
    alert(e?.message || '操作失败')
  }
}

async function handleUnban(id: string) {
  if (!confirm('确定要解除封禁吗？')) return
  try {
    await adminApi.unbanUser(id)
    fetchUsers()
  } catch (e: any) {
    alert(e?.message || '操作失败')
  }
}

// 文章管理
const articles = ref<any[]>([])
const articlePage = ref(1)
const articleSize = 20
const articleTotal = ref(0)

async function fetchArticles() {
  try {
    const res: any = await adminApi.getArticles(articlePage.value, articleSize)
    articles.value = res.data || []
    articleTotal.value = res.pagination?.total || 0
  } catch {
    alert('获取文章列表失败')
  }
}

function changeArticlePage(page: number) {
  articlePage.value = page
  fetchArticles()
}

async function handleDeleteArticle(id: string) {
  if (!confirm('确定要删除该文章吗？')) return
  try {
    await adminApi.deleteArticle(id)
    fetchArticles()
  } catch (e: any) {
    alert(e?.message || '操作失败')
  }
}

// 评论管理
const comments = ref<any[]>([])
const commentPage = ref(1)
const commentSize = 20
const commentTotal = ref(0)

async function fetchComments() {
  try {
    const res: any = await adminApi.getComments(commentPage.value, commentSize)
    comments.value = res.data || []
    commentTotal.value = res.pagination?.total || 0
  } catch {
    alert('获取评论列表失败')
  }
}

function changeCommentPage(page: number) {
  commentPage.value = page
  fetchComments()
}

async function handleDeleteComment(id: string) {
  if (!confirm('确定要删除该评论吗？')) return
  try {
    await adminApi.deleteComment(id)
    fetchComments()
  } catch (e: any) {
    alert(e?.message || '操作失败')
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

onMounted(() => {
  fetchUsers()
})
</script>

<style scoped>
.admin-page {
  max-width: 1100px;
  margin: 0 auto;
  padding: 24px;
}

h1 {
  font-size: 24px;
  margin-bottom: 20px;
}

.tabs {
  display: flex;
  gap: 0;
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
  transition: color 0.2s, border-color 0.2s;
}

.tabs button.active {
  color: #3498db;
  border-bottom-color: #3498db;
}

.tabs button:hover {
  color: #3498db;
}

.tab-content {
  min-height: 300px;
}

h2 {
  font-size: 18px;
  margin-bottom: 16px;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th,
.data-table td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid #eee;
  font-size: 14px;
}

.data-table th {
  background: #f9f9f9;
  font-weight: 600;
  color: #333;
}

.badge-user { color: #666; }
.badge-admin { color: #e67e22; font-weight: 600; }
.badge-active { color: #27ae60; }
.badge-banned { color: #e74c3c; font-weight: 600; }
.badge-deleted { color: #e74c3c; }
.badge-draft { color: #95a5a6; }

.col-title { max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.btn-ban {
  padding: 4px 14px;
  background: #e74c3c;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.btn-ban:hover { background: #c0392b; }

.btn-unban {
  padding: 4px 14px;
  background: #27ae60;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.btn-unban:hover { background: #219a52; }

.btn-delete {
  padding: 4px 14px;
  background: #e74c3c;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.btn-delete:hover { background: #c0392b; }

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

.pagination button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pagination span { font-size: 14px; color: #666; }

.placeholder {
  color: #999;
  font-size: 15px;
  padding: 40px 0;
  text-align: center;
}
</style>
