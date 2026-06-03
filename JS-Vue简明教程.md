# JS-Vue 简明教程

> 记录项目中常见的 JavaScript/Vue3 写法困惑。

## 1. 箭头函数

### 写法等效

```js
// 箭头函数
(error) => { ... }

// 等效于传统匿名函数
function(error) { ... }

// 一个参数时可省略括号
error => { ... }

// 单行返回值可省略 {} 和 return
(response) => response.data

// 等效于
(response) => { return response.data }
```

---

## 2. Axios 响应拦截器

```typescript
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error.response?.data || error)
  }
)
```

### `(response) => response.data` — 拿到的数据是什么？

axios 原始响应对象嵌套了两层：

```
axios 的 response 对象：
├── status: 200                    // HTTP 状态码
├── headers: {...}                 // 响应头
├── config: {...}                  // 请求配置
└── data: {                        // ← 后端的 JSON 体
      code: 200,
      message: "success",
      data: {                      // ← 真正的业务数据
        token: "xxx",
        list: [...],
      }
    }
```

`(response) => response.data` 剥掉外层 axios 包装，返回的是：

```js
{
  code: 200,
  message: "success",
  data: { token: "xxx", list: [...] }   // 业务数据在这一层
}
```

### 拿到后怎么用？

```typescript
const res = await userApi.login({ username, password })
// res 已经被拦截器剥了一层，直接是后端 JSON

token.value = res.data.token    // 再取一层 data，拿到业务数据
```

完整数据流：

```
后端返回 JSON
  → axios 包一层 (多了 status/headers)
    → 拦截器剥掉 axios 包装 (response => response.data)
      → 组件拿到后端 JSON (res)
        → res.data.xxx 取业务数据
```

### `(error) => { ... }` — error 从哪来？

error 是 **axios 传进来的**，不是你定义的。这是一个**回调函数**——你告诉 axios "失败时调这个函数"，axios 在失败时把错误对象传给你。

```typescript
// 类比：跟 addEventListener 一模一样
button.addEventListener('click', (event) => {
  // event 不是你自己定义的，是浏览器在点击时传给你的
})

// 同理：
api.interceptors.response.use(
  (response) => {},  // 成功时 axios 把响应传给你
  (error) => {}      // 失败时 axios 把错误传给你
)
```

`?.` 的含义：

```typescript
error.response?.status
// 等价于
error.response !== null && error.response !== undefined
  ? error.response.status
  : undefined
// 如果 response 不存在，直接返回 undefined，不抛异常
```

### 最后一行 `Promise.reject(...)` 是必要的

```typescript
return Promise.reject(error.response?.data || error)
```

把错误继续往外抛，不然调用方的 `catch` 收不到错误。

---

## 3. 导出：`export default` 与 `export const`

文件中有两种导出方式：

```typescript
// 默认导出 — 原始的 axios 实例
export default api

// 命名导出 — 封装好的分组方法（日常主要用这些）
export const userApi = { login(){}, register(){} }
export const articleApi = { getList(){}, create(){} }
// ...
```

### `export const userApi = { ... }`

就是一个按功能模块分组的**普通 JavaScript 对象**：

```typescript
export const userApi = {
  login(data) {                              // 对象的方法
    return api.post('/users/login', data)
  },
  register(data) {
    return api.post('/users/register', data)
  },
  getProfile(id) {
    return api.get(`/users/${id}`)
  },
}
```

为什么这样写？图的是 `模块.方法()` 的清晰结构，避免十几个散装函数：

```typescript
// ✅ 分组后，调用一目了然
import { userApi } from '@/api'
userApi.login(...)

// ❌ 如果散开，import 越写越长
import { login, register, getProfile, getArticles, createArticle, ... } from '@/api'
```

### `export default api`

`api` 是前面创建的 axios 实例。默认导出后，别的文件可以随意起名导入：

```typescript
// 默认导出 → 导入时随意起名
import http from '@/api'           // 叫 http
import axiosInstance from '@/api'  // 叫 axiosInstance 也行

// 命名导出 → 必须用原名
import { userApi, articleApi } from '@/api'
```

默认导出是给需要**直接发请求**的场景用，不走封装好的分组方法：

```typescript
import http from '@/api'

// 比如调一个没封装到 userApi 里的边缘接口
const res = await http.post('/users/reset-password', { email: 'xxx' })
```

### `import { ... } from '...'` — 导入

`export` 是往外给，`import` 是往里拿：

```typescript
// 别人在这些地方 export 了东西：
// pinia 包里：        export function defineStore() { ... }
// vue 包里：          export function ref() { ... }
//                      export function computed() { ... }
// @/api/index.ts 里： export const userApi = { ... }

// 我的文件里 import 进来：
import { defineStore } from 'pinia'     // 从 npm 包导入
import { ref, computed } from 'vue'     // 从 npm 包导入，一次导入多个
import { userApi } from '@/api'         // 从自己的文件导入
```

规律：

```
import { 别人的导出名 } from '来源路径'
        ↑                          ↑
    export const 的名字       npm 包名 或 文件路径
```

- `{ }` 里的名字必须和对方 `export` 的名字完全一致
- `from` 后面：不带 `./` 或 `/` 的是 npm 包名，带路径的是自己的文件
- `@/` 是 `src/` 的别名，在 `vite.config.ts` 里配置的，等价于 `../../api`

**命名导入 vs 默认导入 vs 混合：**

```typescript
// 命名导入 — 按需取用
import { ref, computed, onMounted } from 'vue'

// 默认导入 — 拿对方的 export default
import api from '@/api'       // 对应 export default api

// 混合用
import api, { userApi } from '@/api'
//     ↑ 默认    ↑ 命名
```

### 两种导出的区别

| | `export default` | `export const` |
|---|---|---|
| 一个文件能写几个 | 1 个 | 任意多个 |
| 导入名字 | 随意起名 | 必须用原名，用 `{}` 包裹 |
| 适用场景 | 导出"主角"（核心实例） | 导出多个工具、方法、常量 |
| 示例 | `import http from '@/api'` | `import { userApi } from '@/api'` |

---

## 4. JavaScript 常用语法速查

| 语法 | 含义 | 示例 |
|------|------|------|
| `?.` | 可选链，左边的值为 null/undefined 时不报错 | `error.response?.status` |
| `\|\|` | 逻辑或，左边为 falsy 时取右边 | `error.message \|\| '登录失败'` |
| `??` | 空值合并，只有 null/undefined 时取右边 | `count ?? 0` |
| `!` | 非空断言，告诉 TS "我确定这不是 null" | `articleId!` |
| `...` | 展开运算符 | `[...arr1, ...arr2]` / `{...obj1, key: val}` |
| `=>` | 箭头函数 | `(x) => x * 2` |
| `` `...${}` `` | 模板字符串 | `` `/articles/${id}` `` |
| `const { x } = obj` | 解构赋值 | `const { data } = response` |

## 5. Vue3 常用语法速查

| 语法 | 作用 |
|------|------|
| `ref(val)` | 创建响应式变量，JS 中 `.value` 访问，模板中自动解包 |
| `computed(() => ...)` | 计算属性，依赖变化时自动重新计算 |
| `onMounted(() => ...)` | 组件挂载到 DOM 后执行 |
| `watch(source, callback)` | 监听某个值的变化 |
| `v-model="x"` | 双向绑定表单输入 |
| `v-if / v-else / v-for` | 条件渲染 / 列表渲染 |
| `@click / @submit.prevent` | 事件绑定 |
| `:prop="val"` | 动态属性绑定（简写 `v-bind:`） |
| `<script setup>` | 语法糖，自动暴露变量给模板 |
