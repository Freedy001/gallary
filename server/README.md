# 图片管理系统后端

基于 Go + Gin + GORM + PostgreSQL 的图片管理系统后端服务。

## 功能特性

✅ **核心功能**
- 图片上传/下载（支持去重）
- 图片列表查询（分页）
- 图片详情查看
- 图片删除
- 图片搜索（多条件）

✅ **技术特性**
- SHA256 哈希去重
- EXIF 信息自动提取
- 多存储适配器支持（当前实现本地存储）
- JWT 认证（可选）
- RESTful API
- 优雅关闭

## 技术栈

- **框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL
- **配置**: Viper
- **日志**: Zap
- **认证**: JWT

## 目录结构

```
server/
├── cmd/
│   └── main.go                 # 程序入口
├── config/
│   └── config.go              # 配置结构
├── internal/
│   ├── handler/               # HTTP 处理器
│   ├── service/              # 业务逻辑层
│   ├── repository/           # 数据访问层
│   ├── model/                # 数据模型
│   ├── middleware/           # 中间件
│   ├── storage/              # 存储适配器
│   ├── utils/                # 工具函数
│   └── router/               # 路由配置
├── pkg/                      # 公共包
│   ├── logger/               # 日志包
│   └── database/             # 数据库连接
├── docs/
│   └── schema.sql            # 数据库表结构
├── config.yaml               # 配置文件
└── go.mod
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- PostgreSQL 13+

### 2. 安装依赖

```bash
cd server
go mod download
```

### 3. 配置数据库

创建 PostgreSQL 数据库：

```sql
CREATE DATABASE gallery;
```

执行数据库表结构（可选，程序会自动迁移）：

```bash
psql -U postgres -d gallery -f docs/schema.sql
```

### 4. 配置文件

编辑 `config.yaml`：

```yaml
database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "your_password"
  database: "gallery"

admin:
  # 留空表示无需认证
  password: ""
```

### 5. 运行程序

```bash
go run cmd/main.go
```

服务器将在 `http://localhost:8080` 启动。

## API 接口文档

### 认证接口

#### 登录（如果启用认证）

```http
POST /api/auth/login
Content-Type: application/json

{
  "password": "your_password"
}
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGc...",
    "expires_in": 604800
  }
}
```

#### 检查认证状态

```http
GET /api/auth/check
```

### 图片接口

所有图片接口需要在请求头中携带 token（如果启用认证）：

```
Authorization: Bearer <token>
```

#### 上传图片

```http
POST /api/images/upload
Content-Type: multipart/form-data

file: <binary>
```

响应：
```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "id": 1,
    "uuid": "xxx-xxx-xxx",
    "original_name": "photo.jpg",
    "file_size": 1024000,
    "file_hash": "sha256...",
    "width": 1920,
    "height": 1080,
    "taken_at": "2024-01-01T10:00:00Z",
    "latitude": 39.9042,
    "longitude": 116.4074,
    "camera_model": "Canon EOS 5D Mark IV",
    ...
  }
}
```

**去重说明**：如果上传的图片 hash 已存在，将返回已存在的图片信息，不会重复存储。

#### 获取图片列表

```http
GET /api/images?page=1&page_size=20
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

#### 获取图片详情

```http
GET /api/images/{id}
```

#### 下载图片

```http
GET /api/images/{id}/download
```

#### 删除图片

```http
DELETE /api/images/{id}
```

#### 搜索图片

```http
GET /api/search?keyword=vacation&start_date=2024-01-01&location=Beijing&page=1
```

支持的查询参数：
- `keyword`: 关键词（搜索文件名）
- `start_date`: 开始日期
- `end_date`: 结束日期
- `location`: 地点名称
- `camera_model`: 相机型号
- `tags`: 标签ID列表（逗号分隔）
- `page`: 页码
- `page_size`: 每页数量

### 健康检查

```http
GET /health
```

## 图片去重机制

系统使用 SHA256 哈希算法对上传的图片进行去重：

1. 上传图片时，先计算文件的 SHA256 哈希值
2. 查询数据库中是否存在相同 hash 的图片
3. 如果存在，直接返回已有图片信息，不重复存储
4. 如果不存在，则保存新图片

这样可以有效节省存储空间，避免重复上传相同的图片。

## 存储配置

当前支持本地存储，图片将保存在 `storage/images/` 目录下，按日期分目录存储：

```
storage/images/
├── 2024/
│   ├── 01/
│   │   ├── 01/
│   │   │   ├── uuid1.jpg
│   │   │   └── uuid2.png
│   │   └── 02/
│   └── 02/
└── ...
```

访问URL格式：`http://localhost:8080/static/images/2024/01/01/uuid1.jpg`

## 认证配置

### 不使用认证

在 `config.yaml` 中将 `admin.password` 留空：

```yaml
admin:
  password: ""
```

### 启用认证

1. 生成 bcrypt 密码哈希：

```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "your_password"
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
```

2. 将哈希值配置到 `config.yaml`：

```yaml
admin:
  password: "$2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

3. 登录获取 token，然后在后续请求中携带 token。

## 开发说明

### 添加新的存储适配器

1. 在 `internal/storage/` 目录创建新文件
2. 实现 `Storage` 接口
3. 在 `cmd/main.go` 中添加初始化逻辑

示例：

```go
type OSSStorage struct {
    // ...
}

func (s *OSSStorage) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
    // 实现OSS上传逻辑
}

// 实现其他接口方法...
```

### 数据库迁移

程序启动时会自动执行数据库迁移，也可以手动执行 SQL 文件：

```bash
psql -U postgres -d gallery -f docs/schema.sql
```

## 常见问题

### 1. 数据库连接失败

检查 `config.yaml` 中的数据库配置是否正确，确保 PostgreSQL 服务正在运行。

### 2. 上传文件大小限制

默认限制为 50MB，可在 `config.yaml` 中修改：

```yaml
image:
  max_size: 104857600  # 100MB (字节)
```

### 3. 支持的图片格式

默认支持：JPEG, PNG, GIF, WebP, HEIC

可在 `config.yaml` 中修改：

```yaml
image:
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "image/webp"
    - "image/heic"
```

## 后续扩展建议

- [ ] 标签管理功能
- [ ] 分享功能
- [ ] 缩略图生成
- [ ] 对象存储支持（OSS/S3/MinIO）
- [ ] 图片编辑功能
- [ ] 批量操作
- [ ] 地理位置反查
- [ ] 相似图片搜索

## License

MIT
