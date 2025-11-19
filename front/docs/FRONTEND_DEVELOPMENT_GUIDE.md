# 图片管理系统前端开发指南

## 项目概述

这是一个基于 **Vue 3 + TypeScript + Vite + TailwindCSS** 的现代化图片管理系统前端应用。后端采用 Go + Gin + PostgreSQL 技术栈，提供了完整的图片上传、检索、管理功能。

## 技术栈

- **框架**: Vue 3 (Composition API)
- **语言**: TypeScript
- **构建工具**: Vite
- **样式**: TailwindCSS 3.x
- **状态管理**: Pinia (推荐)
- **路由**: Vue Router 4
- **HTTP 客户端**: Axios
- **UI 增强**:
  - HeadlessUI (无样式组件)
  - Heroicons (图标库)
  - VueUse (工具库)

## 设计理念

### 🎨 创意方向

1. **iPhone 相册风格网格布局**: 整齐的网格布局，紧凑排列，类似 iOS 照片应用
2. **双指缩放预览**: 支持捏合缩放（pinch-to-zoom）查看大图，流畅的缩放动画
3. **沉浸式浏览体验**: 点击图片全屏查看，支持手势滑动切换、双击缩放
4. **智能搜索界面**: 命令面板式搜索，支持多条件筛选
5. **极简设计风格**: 白色背景 + 细边框，突出图片内容本身 
6. **动态主题色**: 根据当前浏览图片的主色调自适应界面配色.
7. **3D 卡片效果**: 鼠标悬停时的视差效果和阴影变化
6. **流畅动画**: 使用 Tailwind 动画 + CSS Transform 实现极致流畅体验

### 🚀 核心特性

- ✨ **无缝上传**: 拖拽上传 + 粘贴上传 + 批量上传
- 🔍 **高级搜索**: 按时间、地点、相机型号、EXIF 信息搜索
- 📍 **地图视图**: 基于 GPS 坐标的地图标注展示
- 📊 **EXIF 可视化**: 以图表形式展示拍摄参数统计
- 🏷️ **智能标签**: AI 自动标签推荐（预留接口）
- 📱 **响应式设计**: 完美适配移动端/平板/桌面端
- ⚡ **虚拟滚动**: 处理海量图片列表
- 🎭 **骨架屏**: 优雅的加载状态

## 项目结构

```
front/
├── public/                    # 静态资源
├── src/
│   ├── assets/               # 资源文件
│   │   ├── images/          # 图片资源
│   │   └── styles/          # 全局样式
│   │       └── main.css     # Tailwind 入口
│   ├── components/           # 组件
│   │   ├── common/          # 通用组件
│   │   │   ├── Button.vue
│   │   │   ├── Input.vue
│   │   │   ├── Modal.vue
│   │   │   ├── Loading.vue
│   │   │   └── DropZone.vue
│   │   ├── layout/          # 布局组件
│   │   │   ├── Header.vue
│   │   │   ├── Sidebar.vue
│   │   │   └── Footer.vue
│   │   ├── image/           # 图片相关组件
│   │   │   ├── ImageCard.vue
│   │   │   ├── ImageGrid.vue
│   │   │   ├── ImageLightbox.vue
│   │   │   ├── ImageUploader.vue
│   │   │   └── ExifPanel.vue
│   │   └── search/          # 搜索相关组件
│   │       ├── SearchBar.vue
│   │       ├── SearchPanel.vue
│   │       └── FilterChips.vue
│   ├── views/                # 页面视图
│   │   ├── Home.vue         # 主页
│   │   ├── Gallery.vue      # 图片画廊
│   │   ├── Upload.vue       # 上传页面
│   │   ├── Search.vue       # 搜索页面
│   │   ├── MapView.vue      # 地图视图
│   │   ├── Timeline.vue     # 时间线视图
│   │   └── Login.vue        # 登录页面
│   ├── api/                  # API 接口
│   │   ├── http.ts          # Axios 封装
│   │   ├── auth.ts          # 认证接口
│   │   └── image.ts         # 图片接口
│   ├── stores/               # Pinia 状态管理
│   │   ├── auth.ts          # 认证状态
│   │   ├── image.ts         # 图片状态
│   │   └── ui.ts            # UI 状态
│   ├── composables/          # 组合式函数
│   │   ├── useAuth.ts       # 认证逻辑
│   │   ├── useUpload.ts     # 上传逻辑
│   │   ├── useInfiniteScroll.ts
│   │   └── useImageViewer.ts
│   ├── types/                # TypeScript 类型定义
│   │   ├── image.ts
│   │   ├── api.ts
│   │   └── common.ts
│   ├── utils/                # 工具函数
│   │   ├── format.ts        # 格式化工具
│   │   ├── date.ts          # 日期工具
│   │   ├── file.ts          # 文件处理
│   │   └── color.ts         # 颜色提取
│   ├── router/               # 路由配置
│   │   └── index.ts
│   ├── App.vue              # 根组件
│   └── main.ts              # 入口文件
├── tailwind.config.js       # Tailwind 配置
├── vite.config.ts           # Vite 配置
├── tsconfig.json            # TypeScript 配置
└── package.json             # 依赖管理
```

## API 接口集成

### 基础配置

**baseURL**: `http://localhost:9099` (根据后端实际端口调整)

### 接口列表

#### 1. 认证接口

```typescript
// POST /api/auth/login
interface LoginRequest {
  password: string;
}

interface LoginResponse {
  code: number;
  message: string;
  data: {
    token: string;
    expires_in: number;
  };
}

// GET /api/auth/check
interface CheckAuthResponse {
  code: number;
  message: string;
  data: {
    authenticated: boolean;
  };
}
```

#### 2. 图片管理接口

```typescript
// POST /api/images/upload
// Content-Type: multipart/form-data
// Form field: file

// GET /api/images?page=1&page_size=20
interface ImageListRequest {
  page?: number;
  page_size?: number;
}

// GET /api/images/:id
// DELETE /api/images/:id
// GET /api/images/:id/download

interface Image {
  id: number;
  uuid: string;
  original_name: string;
  storage_path: string;
  storage_type: string;
  file_size: number;
  file_hash: string;
  mime_type: string;
  width: number;
  height: number;
  taken_at: string;
  latitude: number | null;
  longitude: number | null;
  location_name: string | null;
  camera_model: string | null;
  camera_make: string | null;
  aperture: string | null;
  shutter_speed: string | null;
  iso: number | null;
  focal_length: string | null;
  tags: Tag[];
  metadata: Metadata[];
  created_at: string;
  updated_at: string;
}

interface Tag {
  id: number;
  name: string;
  color: string;
}

interface Metadata {
  id: number;
  meta_key: string;
  meta_value: string;
  value_type: string;
}

interface ImageListResponse {
  code: number;
  message: string;
  data: {
    list: Image[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
  };
}
```

#### 3. 搜索接口

```typescript
// GET /api/search
interface SearchRequest {
  keyword?: string;         // 关键词
  start_date?: string;      // YYYY-MM-DD
  end_date?: string;        // YYYY-MM-DD
  location?: string;        // 地点名称
  camera_model?: string;    // 相机型号
  page?: number;
  page_size?: number;
}
```

#### 4. 静态资源访问

```
http://localhost:9099/static/images/{storage_path}
```

### 响应格式

所有接口统一返回格式：

```typescript
interface ApiResponse<T = any> {
  code: number;      // 0 表示成功
  message: string;   // 响应消息
  data?: T;          // 响应数据
}

// 错误码
enum ErrorCode {
  SUCCESS = 0,
  BAD_REQUEST = 400,
  UNAUTHORIZED = 401,
  FORBIDDEN = 403,
  NOT_FOUND = 404,
  SERVER_ERROR = 500
}
```

## 核心功能实现指南

### 1. 认证系统

**实现要点**：
- 登录后将 token 存储在 `localStorage`
- Axios 拦截器自动添加 `Authorization: Bearer <token>` 头
- Token 过期自动跳转登录页
- 支持"记住密码"功能

### 2. 图片上传

**关键特性**：
- 拖拽上传
- 粘贴上传（监听 `paste` 事件）
- 批量上传（多文件选择）
- 上传进度显示
- 上传前预览
- 文件类型/大小验证

### 3. iPhone 相册风格网格布局

**设计特点**：
- 整齐的正方形网格
- 图片紧凑排列，间距极小（1-2px）
- 自适应响应式列数
- 图片裁剪为正方形（`object-fit: cover`）
- 支持缩放，自适应调整图片大小以及数量

### 4. 图片查看器（支持缩放）

**核心功能**：
- ✨ **双指捏合缩放**: 移动端支持 pinch-to-zoom
- 🖱️ **鼠标滚轮缩放**: 桌面端支持滚轮缩放
- 👆 **双击缩放**: 双击放大/还原
- 🔄 **平滑过渡**: 缩放和拖动都有流畅的动画
- ⬅️➡️ **左右切换**: 支持键盘/手势切换上下张
- 📊 **EXIF 信息面板**: 显示拍摄参数
- 🌑 **黑色背景**: 沉浸式全屏体验

**推荐库**：
- `@vueuse/gesture` - 手势识别（pinch、pan、tap）
- `@vueuse/core` - 实用 Composables（useKeyPress、useSwipe）
- 或使用 `vue-picture-swipe` / `v-viewer`

### 5. 高级搜索

**UI 设计**：
- 命令面板风格（Cmd+K 唤起）
- 多条件筛选器
- 搜索历史记录
- 智能建议（相机型号、地点自动补全）

### 6. 地图视图

**实现方案**：
- 使用 Leaflet.js + OpenStreetMap
- 或使用高德/百度地图 API

**功能**：
- 标注所有有 GPS 坐标的图片
- 点击标注显示图片预览
- 聚合显示（相近的图片合并为一个标记）

### 7. 时间线视图

**布局方式**：
- 按年/月/日分组
- 垂直时间线 + 横向滚动图片
- 标记重要日期（节假日、旅行等）

## TailwindCSS 创意设计指南

### 创意组件

#### 1. 3D 卡片悬停效果
#### 2. 毛玻璃导航栏
#### 3. 加载骨架屏
#### 4. 上传进度条



## 参考资源

### 设计灵感
- [Unsplash](https://unsplash.com) - 图片展示
- [Pinterest](https://pinterest.com) - 瀑布流布局
- [Google Photos](https://photos.google.com) - 时间线 + 搜索

### 技术文档
- [Vue 3 文档](https://cn.vuejs.org/)
- [TailwindCSS 文档](https://tailwindcss.com/)
- [VueUse 文档](https://vueuse.org/)
- [HeadlessUI 文档](https://headlessui.com/)

### 组件库参考
- [TailwindUI](https://tailwindui.com/) - 付费组件
- [HyperUI](https://www.hyperui.dev/) - 免费组件
- [Flowbite](https://flowbite.com/) - 免费组件库

## 常见问题

### Q1: 如何处理认证？
**A**: 如果后端配置了 `admin.password`，需要登录获取 token；如果未配置，可跳过登录直接访问。

### Q2: 图片去重如何体现？
**A**: 后端已实现 SHA256 去重，前端上传相同图片时会返回已存在的图片信息。可在 UI 上提示"图片已存在"。

### Q3: 如何优化大图加载？
**A**:
1. 列表使用缩略图（可考虑让后端提供缩略图 API）
2. 使用 `loading="lazy"` 属性
3. 使用 Intersection Observer 懒加载

### Q4: 移动端手势支持？
**A**: 使用 `@vueuse/gesture` 或 `hammerjs` 实现滑动、缩放手势。

## 总结

这是一个充满创意空间的项目！后端已经提供了完善的 API，前端的重点在于：

1. **视觉设计**: 利用 TailwindCSS 创造独特的视觉体验
2. **交互体验**: 流畅的动画和手势操作
3. **性能优化**: 虚拟滚动、懒加载、缓存策略
4. **功能完整**: 上传、搜索、浏览、管理一体化

开发过程中保持创造性思维，打破常规设计，实现一个令人眼前一亮的现代化图片管理系统！

---

**祝开发顺利！有任何问题欢迎查阅后端 API 文档或联系后端团队。**
