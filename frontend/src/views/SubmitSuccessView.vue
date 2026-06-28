<template>
  <div class="submit-success">
    <div class="success-card">
      <div class="success-icon">&#10003;</div>
      <h2>提交成功</h2>
      <p>文章已提交审核，请等待管理员审核。</p>
      <div class="actions">
        <button class="btn-primary" @click="$router.push('/')">首页</button>
        <button class="btn-secondary" @click="$router.push('/profile')">个人中心</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.submit-success {
  max-width: 500px;
  margin: 80px auto;
  padding: 0 16px;
}

.success-card {
  background: #fff;
  border-radius: 12px;
  padding: 48px 32px;
  text-align: center;
}

.success-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: #27ae60;
  color: #fff;
  font-size: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 20px;
}

.success-card h2 {
  font-size: 22px;
  margin: 0 0 8px;
  color: #222;
}

.success-card p {
  color: #888;
  margin: 0 0 28px;
  font-size: 14px;
}

.actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.btn-primary {
  padding: 10px 28px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
}

.btn-primary:hover {
  background: #2980b9;
}

.btn-secondary {
  padding: 10px 28px;
  background: #fff;
  color: #555;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
}

.btn-secondary:hover {
  border-color: #3498db;
  color: #3498db;
}
</style>
