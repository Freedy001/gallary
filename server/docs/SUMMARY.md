# 图片管理系统后端 - 项目总结

## 项目概述

这是一个基于 Go 语言的图片管理系统后端，采用 **Gin + GORM + PostgreSQL** 技术栈，实现了图片的上传、存储、检索、下载等核心功能，并支持基于 SHA256 哈希的图片去重机制。

## 核心功能

### 1. 图片管理
- ✅ 图片上传（支持批量）
- ✅ 图片下载
- ✅ 图片列表查询（分页）
- ✅ 图片详情查看
- ✅ 图片删除（软删除）
- ✅ **SHA256 哈希去重**（核心特性）

### 2. 图片搜索
- ✅ 按文件名关键词搜索
- ✅ 按拍摄时间范围搜索
- ✅ 按地点搜索
- ✅ 按相机型号搜索
- ✅ 按标签搜索（数据库已支持）
- ✅ 多条件组合搜索

### 3. 元数据管理
- ✅ 自动提取 EXIF 信息
  - 拍摄时间
  - GPS 坐标（经纬度）
  - 相机信息（品牌、型号）
  - 拍摄参数（光圈、快门、ISO、焦距）
- ✅ 图片尺寸信息
- ✅ 自定义元数据（数据库已支持）

### 4. 存储系统
- ✅ 本地存储（已实现）
- ✅ 可扩展存储适配器架构
- ⏳ 对象存储（OSS/S3/MinIO）预留接口

### 5. 认证授权
- ✅ JWT 认证（可选）
- ✅ Bcrypt 密码加密
- ✅ 灵活的认证配置

## 技术架构

### 分层架构

```
┌─────────────────────────────────────────┐
│           HTTP Handler Layer            │  API 接口层
├─────────────────────────────────────────┤
│           Service Layer                 │  业务逻辑层
├─────────────────────────────────────────┤
│          Repository Layer               │  数据访问层
├─────────────────────────────────────────┤
│           Model Layer                   │  数据模型层
└─────────────────────────────────────────┘
         │                    │
         ↓                    ↓
┌──────────────┐    ┌──────────────┐
│  PostgreSQL  │    │   Storage    │
│   Database   │    │   Adapter    │
└──────────────┘    └──────────────┘
```

### 核心模块

#### 1. Handler 层（API 接口）
- `auth_handler.go`: 认证相关接口
- `image_handler.go`: 图片管理接口

#### 2. Service 层（业务逻辑）
- `image_service.go`: 图片业务逻辑
  - 上传处理
  - 去重检查
  - EXIF 提取
  - 存储协调

#### 3. Repository 层（数据访问）
- `image_repository.go`: 图片数据访问
  - CRUD 操作
  - 复杂查询
  - 事务处理

#### 4. Storage 层（存储适配器）
- `storage.go`: 存储接口定义
- `local.go`: 本地存储实现

#### 5. Model 层（数据模型）
- `image.go`: 图片模型
- `tag.go`: 标签模型
- `metadata.go`: 元数据模型
- `share.go`: 分享模型

## 数据库设计

### 表结构概览

```sql
images              -- 图片主表
├── id (PK)
├── uuid (Unique)
├── file_hash (Unique) -- 用于去重
├── original_name
├── storage_path
├── file_size
├── mime_type
├── width, height
├── EXIF 字段（taken_at, latitude, longitude, camera_model, etc.）
└── 时间戳（created_at, updated_at, deleted_at）

tags                -- 标签表
├── id (PK)
├── name (Unique)
└── color

image_tags          -- 图片标签关联表
├── image_id (FK)
└── tag_id (FK)

image_metadata      -- 自定义元数据表
├── image_id (FK)
├── meta_key
├── meta_value
└── value_type

shares              -- 分享表
├── id (PK)
├── share_code (Unique)
├── password
├── expire_at
└── 统计字段

share_images        -- 分享图片关联表
├── share_id (FK)
└── image_id (FK)
```

### 关键索引

- `images.file_hash`: 唯一索引（去重）
- `images.taken_at`: 普通索引（时间查询）
- `images.latitude, longitude`: 复合索引（地理查询）
- `images.deleted_at`: 普通索引（软删除）

## 核心特性详解

### 1. 图片去重机制

**实现原理**：

1. 文件上传后，先保存到临时文件
2. 计算文件的 SHA256 哈希值
3. 查询数据库 `file_hash` 字段
4. 如果哈希已存在，返回已有图片信息
5. 如果不存在，继续上传流程

**代码位置**: `internal/service/image_service.go:Upload()`

**优势**：
- 节省存储空间
- 防止重复上传
- 保持数据一致性

### 2. EXIF 信息提取

**支持字段**：
- 拍摄时间（DateTime）
- GPS 坐标（Latitude/Longitude）
- 相机信息（Make/Model）
- 拍摄参数（Aperture/Shutter/ISO/Focal Length）

**代码位置**: `internal/utils/exif.go`

**技术**: 使用 `github.com/rwcarlsen/goexif` 库

### 3. 存储适配器模式

**接口定义**：
```go
type Storage interface {
    Upload(ctx, file, path) (string, error)
    Download(ctx, path) (io.ReadCloser, error)
    Delete(ctx, path) error
    GetURL(ctx, path) (string, error)
    Exists(ctx, path) (bool, error)
}
```

**已实现**：
- LocalStorage（本地文件系统）

**可扩展**：
- OSSStorage（阿里云 OSS）
- S3Storage（AWS S3）
- MinIOStorage（MinIO）

### 4. 分层架构设计

**职责分离**：
- Handler: 处理 HTTP 请求，参数验证
- Service: 业务逻辑，流程控制
- Repository: 数据访问，SQL 查询
- Model: 数据结构定义

**优势**：
- 代码清晰易维护
- 便于单元测试
- 易于扩展

## 配置管理

### config.yaml 结构

```yaml
server:       # 服务器配置
  host, port, mode

admin:        # 管理员配置
  password    # 留空则无需认证

database:     # 数据库配置
  host, port, username, password, database

storage:      # 存储配置
  default     # 默认存储类型
  local       # 本地存储配置
  oss/s3/minio # 对象存储配置

image:        # 图片配置
  allowed_types, max_size, thumbnail

jwt:          # JWT 配置
  secret, expire_hours

logger:       # 日志配置
  level, format, output

cors:         # CORS 配置
  allow_origins, allow_methods

share:        # 分享配置
  default_expire_hours, code_length
```

## API 接口设计

### RESTful 风格

```
POST   /api/auth/login              # 登录
GET    /api/auth/check              # 检查认证

POST   /api/images/upload           # 上传图片
GET    /api/images                  # 图片列表
GET    /api/images/:id              # 图片详情
DELETE /api/images/:id              # 删除图片
GET    /api/images/:id/download     # 下载图片

GET    /api/search                  # 搜索图片

GET    /health                      # 健康检查
```

### 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

## 已完成功能清单

### ✅ 核心功能
- [x] 图片上传
- [x] 图片下载
- [x] 图片列表
- [x] 图片详情
- [x] 图片删除
- [x] 图片搜索
- [x] SHA256 去重
- [x] EXIF 提取

### ✅ 技术实现
- [x] PostgreSQL 数据库
- [x] GORM ORM 框架
- [x] Gin Web 框架
- [x] Viper 配置管理
- [x] Zap 日志系统
- [x] JWT 认证
- [x] 本地存储
- [x] CORS 支持
- [x] 优雅关闭


## 总结

这是一个设计合理、功能完整的图片管理系统后端。核心功能已全部实现，包括**图片上传、存储、检索、下载和基于哈希的去重机制**。代码结构清晰，采用分层架构，易于维护和扩展。

系统已经可以投入使用，同时预留了丰富的扩展接口，可以根据实际需求逐步添加标签管理、分享功能、对象存储等高级特性。
