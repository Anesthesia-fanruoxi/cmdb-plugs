# File Plugs

安全的文件上传管理服务，支持多文件上传、目录结构保留、自动解压等功能。

## 快速开始

```bash
go mod tidy
go run main.go
```

服务默认运行在 `http://localhost:8083`

## 配置

编辑 `config/config.yaml`：

```yaml
server:
  port: 8083

storage:
  max_file_size: 10485760  # 10MB
  paths:
    - key: "logs"
      path: "./uploads/logs"
      max_file_size: 5242880
      allowed_types: [".log", ".txt"]
      auto_unzip: false

    - key: "images"
      path: "./uploads/images"
      allowed_types: [".jpg", ".png", ".gif"]
      auto_unzip: false

    - key: "packages"
      path: "./uploads/packages"
      allowed_types: [".zip", ".rar", ".tar.gz"]
      auto_unzip: true
```

## 项目结构

```
file-plugs/
├── main.go              # 主入口
├── config/
│   ├── config.go        # 配置加载
│   └── config.yaml      # 配置文件
├── common/
│   └── storage.go       # 文件存储
└── api/
    ├── upload.go        # 文件上传
    └── list.go          # 文件列表与路径列表
```

## API 接口

### 文件上传

**接口地址:** `POST /api/upload`

**请求参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 上传路径标识 |
| path | string | 否 | 子目录路径 |
| file | file | 是 | 上传的文件 |
| filePath | string | 否 | 文件相对路径（保留目录结构） |

**单文件上传:**

```bash
curl -X POST http://localhost:8083/api/upload \
  -F "key=logs" \
  -F "file=@app.log"
```

**多文件上传:**

```bash
curl -X POST http://localhost:8083/api/upload \
  -F "key=images" \
  -F "file=@photo1.jpg" \
  -F "file=@photo2.jpg"
```

**保留目录结构上传:**

```bash
curl -X POST http://localhost:8083/api/upload \
  -F "key=logs" \
  -F "file=@app.log" \
  -F "filePath=2025/01/19/app.log"
```

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "key": "logs",
    "files": ["2025/01/19/app.log"],
    "count": 1,
    "size": 1024,
    "errors": []
  }
}
```

### 文件列表

**接口地址:** `GET /api/list?key=logs`

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "files": [
      {
        "name": "app.log",
        "path": "2025/01/19/app.log",
        "size": 1024,
        "modified": "2025-01-19T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

### 路径列表

**接口地址:** `GET /api/keys`

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "keys": [
      {
        "key": "logs",
        "path": "./uploads/logs",
        "allowed_types": [".log", ".txt"]
      }
    ]
  }
}
```

## 安全特性

- 路径验证：禁止根目录和当前目录
- 类型限制：可配置允许的文件类型
- 大小限制：可配置最大文件大小
- 路径穿越防护
