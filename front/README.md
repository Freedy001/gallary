# 图片管理系统前端

基于 Vue 3 + TypeScript + Vite + TailwindCSS v4 的现代化图片管理系统前端应用。

## 技术栈

- **框架**: Vue 3 (Composition API)
- **语言**: TypeScript
- **构建工具**: Vite
- **样式**: TailwindCSS 4.x
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **HTTP 客户端**: Axios
- **UI 增强**: HeadlessUI + Heroicons + VueUse

## 快速开始

### 安装依赖

```bash
npm install
```

### 开发模式

```bash
npm run dev
```

前端服务器将运行在 `http://localhost:5173`

### 构建生产版本

```bash
npm run build
```

### 预览生产构建

```bash
npm run preview
```

## 项目结构

```
front/
├── src/
│   ├── api/              # API 接口封装
│   ├── assets/           # 静态资源
│   ├── components/       # Vue 组件
│   ├── composables/      # 组合式函数
│   ├── router/           # 路由配置
│   ├── stores/           # Pinia 状态管理
│   ├── types/            # TypeScript 类型定义
│   ├── utils/            # 工具函数
│   ├── views/            # 页面视图
│   ├── App.vue           # 根组件
│   └── main.ts           # 应用入口
├── index.html
├── vite.config.ts
└── package.json
```

## 核心功能

### 已实现

- ✅ 用户认证（登录/登出）
- ✅ 图片列表展示（iPhone 相册风格网格）
- ✅ 图片上传（拖拽、粘贴、批量上传）
- ✅ 图片搜索（多条件筛选）
- ✅ 响应式设计
- ✅ 加载骨架屏
- ✅ Toast 通知
- ✅ 网格大小调整

### 待实现

- ⏳ 图片查看器（双指缩放、手势操作）
- ⏳ EXIF 信息面板
- ⏳ 地图视图
- ⏳ 时间线视图
- ⏳ 图片删除功能
- ⏳ 虚拟滚动优化

## 环境配置

在项目根目录创建 `.env.development` 文件：

```env
VITE_API_BASE_URL=http://localhost:9099
```

## API 端点

后端 API 运行在 `http://localhost:9099`，支持以下端点：

- `POST /api/auth/login` - 用户登录
- `GET /api/auth/check` - 检查认证状态
- `POST /api/images/upload` - 上传图片
- `GET /api/images` - 获取图片列表
- `GET /api/images/:id` - 获取图片详情
- `DELETE /api/images/:id` - 删除图片
- `GET /api/search` - 搜索图片

## 开发指南

详细的开发指南请参考 [FRONTEND_DEVELOPMENT_GUIDE.md](./docs/FRONTEND_DEVELOPMENT_GUIDE.md)

## 常见问题

### Q: 如何修改后端 API 地址？

A: 修改 `.env.development` 或 `.env.production` 文件中的 `VITE_API_BASE_URL` 变量。

### Q: 如何调整网格列数？

A: 在相册页面点击 `-` 和 `+` 按钮，或在 `stores/ui.ts` 中修改 `gridColumns` 的默认值。

### Q: 支持哪些图片格式？

A: 支持 JPG, PNG, GIF, WEBP, HEIC 格式，单个文件最大 50MB。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT
