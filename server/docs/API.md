# API 接口文档

## 基础信息

- **Base URL**: `http://localhost:8080`
- **认证方式**: JWT Bearer Token（可选）
- **Content-Type**: `application/json`（除上传接口外）

## 认证相关

### 1. 登录

如果在配置文件中设置了管理员密码，需要先登录获取 token。

**请求**

```http
POST /api/auth/login
Content-Type: application/json

{
  "password": "your_password"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 604800
  }
}
```

**说明**
- `expires_in`: token 有效期（秒）
- 登录成功后，在后续请求的 Header 中添加：`Authorization: Bearer <token>`

### 2. 检查认证状态

**请求**

```http
GET /api/auth/check
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "authenticated": true
  }
}
```

## 图片管理

### 1. 上传图片

**请求**

```http
POST /api/images/upload
Content-Type: multipart/form-data
Authorization: Bearer <token>

file: <binary>
```

**响应**

```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "original_name": "photo.jpg",
    "storage_path": "2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "storage_type": "local",
    "file_size": 1024000,
    "file_hash": "abc123...",
    "mime_type": "image/jpeg",
    "width": 1920,
    "height": 1080,
    "taken_at": "2024-01-01T10:00:00Z",
    "latitude": 39.9042,
    "longitude": 116.4074,
    "location_name": null,
    "camera_model": "Canon EOS 5D Mark IV",
    "camera_make": "Canon",
    "aperture": "f/2.8",
    "shutter_speed": "1/125",
    "iso": 400,
    "focal_length": "50.0mm",
    "created_at": "2024-01-15T08:00:00Z",
    "updated_at": "2024-01-15T08:00:00Z"
  }
}
```

**错误响应**

```json
{
  "code": 400,
  "message": "不支持的文件类型: image/bmp"
}
```

```json
{
  "code": 400,
  "message": "文件大小超过限制: 52428800 bytes"
}
```

**去重说明**

如果上传的图片 hash 值已存在，将返回已存在的图片信息：

```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "id": 1,
    "uuid": "existing-uuid",
    ...
  }
}
```

### 2. 获取图片列表

**请求**

```http
GET /api/images?page=1&page_size=20
Authorization: Bearer <token>
```

**查询参数**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "uuid": "...",
        "original_name": "photo.jpg",
        ...
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

### 3. 获取图片详情

**请求**

```http
GET /api/images/{id}
Authorization: Bearer <token>
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | int | 图片ID |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "original_name": "photo.jpg",
    "storage_path": "2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg",
    "file_size": 1024000,
    "file_hash": "abc123...",
    "mime_type": "image/jpeg",
    "width": 1920,
    "height": 1080,
    "taken_at": "2024-01-01T10:00:00Z",
    "latitude": 39.9042,
    "longitude": 116.4074,
    "camera_model": "Canon EOS 5D Mark IV",
    "tags": [
      {
        "id": 1,
        "name": "vacation",
        "color": "#FF5733"
      }
    ],
    "metadata": [
      {
        "id": 1,
        "meta_key": "description",
        "meta_value": "Summer vacation photo",
        "value_type": "string"
      }
    ],
    "created_at": "2024-01-15T08:00:00Z"
  }
}
```

### 4. 删除图片

**请求**

```http
DELETE /api/images/{id}
Authorization: Bearer <token>
```

**响应**

```json
{
  "code": 0,
  "message": "删除成功",
  "data": null
}
```

### 5. 下载图片

**请求**

```http
GET /api/images/{id}/download
Authorization: Bearer <token>
```

**响应**

- Content-Type: 图片的 MIME 类型
- Content-Disposition: attachment; filename=原始文件名
- Body: 图片二进制数据

### 6. 搜索图片

**请求**

```http
GET /api/search?keyword=vacation&start_date=2024-01-01&end_date=2024-12-31&location=Beijing&camera_model=Canon&page=1&page_size=20
Authorization: Bearer <token>
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 关键词（搜索文件名） |
| start_date | string | 否 | 开始日期（YYYY-MM-DD） |
| end_date | string | 否 | 结束日期（YYYY-MM-DD） |
| location | string | 否 | 地点名称 |
| camera_model | string | 否 | 相机型号 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [...],
    "total": 50,
    "page": 1,
    "page_size": 20,
    "total_pages": 3
  }
}
```

## 静态文件访问

本地存储的图片可以通过以下URL直接访问：

```
http://localhost:8080/static/images/{storage_path}
```

例如：
```
http://localhost:8080/static/images/2024/01/15/550e8400-e29b-41d4-a716-446655440000.jpg
```

## 健康检查

**请求**

```http
GET /health
```

**响应**

```json
{
  "status": "ok"
}
```

## 响应码说明

| Code | HTTP Status | 说明 |
|------|-------------|------|
| 0 | 200 | 成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未授权 |
| 403 | 403 | 禁止访问 |
| 404 | 404 | 资源不存在 |
| 500 | 500 | 服务器内部错误 |

## 错误响应格式

```json
{
  "code": 400,
  "message": "错误信息"
}
```

## 使用示例

### cURL 示例

**登录**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"password":"your_password"}'
```

**上传图片**
```bash
curl -X POST http://localhost:8080/api/images/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@/path/to/photo.jpg"
```

**获取图片列表**
```bash
curl -X GET "http://localhost:8080/api/images?page=1&page_size=20" \
  -H "Authorization: Bearer <token>"
```

**搜索图片**
```bash
curl -X GET "http://localhost:8080/api/search?keyword=vacation&start_date=2024-01-01" \
  -H "Authorization: Bearer <token>"
```

**删除图片**
```bash
curl -X DELETE http://localhost:8080/api/images/1 \
  -H "Authorization: Bearer <token>"
```

### JavaScript 示例

```javascript
// 登录
const login = async (password) => {
  const response = await fetch('http://localhost:8080/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password })
  });
  const data = await response.json();
  return data.data.token;
};

// 上传图片
const uploadImage = async (file, token) => {
  const formData = new FormData();
  formData.append('file', file);

  const response = await fetch('http://localhost:8080/api/images/upload', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` },
    body: formData
  });
  return await response.json();
};

// 获取图片列表
const getImages = async (page, pageSize, token) => {
  const response = await fetch(
    `http://localhost:8080/api/images?page=${page}&page_size=${pageSize}`,
    {
      headers: { 'Authorization': `Bearer ${token}` }
    }
  );
  return await response.json();
};

// 搜索图片
const searchImages = async (params, token) => {
  const query = new URLSearchParams(params).toString();
  const response = await fetch(
    `http://localhost:8080/api/search?${query}`,
    {
      headers: { 'Authorization': `Bearer ${token}` }
    }
  );
  return await response.json();
};
```
