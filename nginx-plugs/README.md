# Nginx Plugs

Nginx 配置文件管理插件,集成阿里云 DNS 解析与 SSH 远程 nginx 重载,实现域名解析全流程自动化。

## 快速开始

```bash
# 下载依赖
go mod tidy

# 运行(默认端口 8091)
go run main.go

# 指定端口运行
go run main.go -port 29882
```

服务默认运行在 `http://localhost:8091`

## 配置

加载优先级:**命令行参数 > 环境变量 > 配置文件 > 默认值**

### 命令行参数

| 参数 | 说明 | 示例 |
| --- | --- | --- |
| `-port` | HTTP 服务端口(优先级最高) | `-port 29882` |

### 配置文件

编辑 `config/config.yaml`:

```yaml
server:
  port: 8091

nginx:
  conf_dir: "/etc/nginx/cmdb"  # 配置文件输出目录(自动创建)

aliyun:
  access_key_id: ""      # 阿里云 AccessKey ID
  access_key_secret: ""  # 阿里云 AccessKey Secret
  record_type: "CNAME"   # DNS 记录类型: CNAME 或 A
  record_value: ""       # DNS 记录值

domains:                 # 前端下拉可选的主域名
  - name: "hzlsg.com"
  - name: "xdysh.com"

nginx_cmd:
  reload: "nginx -s reload"  # 远程 nginx 重载命令

ssh_targets:             # 远程 nginx 服务器列表(SSH 重载)
  - name: "nginx-01"
    host: "192.168.1.10"
    port: 22
    user: "root"
    key_path: ""         # SSH 私钥路径(与 password 二选一)
    password: ""         # SSH 密码

log:
  level: "info"
```

> 模板文件固定为 `conf_dir/proxy.conf.template`,首次启动会自动创建默认模板。模板支持以下变量:
>
> | 变量 | 说明 | 示例 |
> | --- | --- | --- |
> | `{{.ServerName}}` | 完整域名 | `testok.hzbxhd.com` |

### 环境变量

```bash
NGINX_PLUGS_PORT=8091          # 服务端口
NGINX_CONF_DIR=/etc/nginx/cmdb # nginx 配置目录
LOG_LEVEL=info                 # 日志级别
ALIYUN_ACCESS_KEY_ID=xxx       # 阿里云 AccessKey ID
ALIYUN_ACCESS_KEY_SECRET=xxx   # 阿里云 AccessKey Secret
ALIYUN_RECORD_TYPE=CNAME       # DNS 记录类型
ALIYUN_RECORD_VALUE=xxx        # DNS 记录值
NGINX_RELOAD_CMD=nginx -s reload # nginx 重载命令
NGINX_DOMAINS=hzlsg.com,xdysh.com # 域名列表(逗号分隔)
```

## nginx 主配置

在 nginx 主配置文件中 include 插件生成的配置:

```nginx
include /etc/nginx/cmdb/*.conf;
```

程序只在 `cmdb/` 子目录内操作,不影响其他 nginx 配置。

## API 接口

所有接口返回标准 JSON 格式:

```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

### 添加解析(全流程)

自动完成:① 添加 DNS 记录 ② 生成 nginx 配置文件 ③ SSH 重载 nginx。

```
GET /api/nginx/add?server_name=testok.baidu.com
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `server_name` | 是 | 完整域名,如 `testok.baidu.com` |

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "server_name": "testok.baidu.com",
    "sub_domain": "testok",
    "domain": "baidu.com",
    "dns_record_id": "1234567",
    "output_file": "/etc/nginx/cmdb/testok_baidu_com.conf",
    "reload_results": [
      { "name": "nginx-01", "host": "192.168.1.10", "status": "success", "error": "" }
    ]
  }
}
```

### 预览配置

读取已生成的配置文件内容(不写入文件)。

```
GET /api/nginx/preview?server_name=testok.baidu.com
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `server_name` | 是 | 完整域名 |

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "server_name": "testok.baidu.com",
    "output_file": "/etc/nginx/cmdb/testok_baidu_com.conf",
    "content": "server { ... }"
  }
}
```

### 删除解析(全流程)

自动完成:① 删除 DNS 记录 ② 删除 nginx 配置文件 ③ SSH 重载 nginx。

```
DELETE /api/nginx/delete?server_name=testok.baidu.com
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `server_name` | 是 | 完整域名 |

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "server_name": "testok.baidu.com",
    "filename": "testok_baidu_com.conf",
    "dns_deleted": true,
    "reload_results": [
      { "name": "nginx-01", "host": "192.168.1.10", "status": "success", "error": "" }
    ]
  }
}
```

### 解析列表

列出配置目录中已生成的 `.conf` 文件。

```
GET /api/nginx/list
```

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "files": [
      {
        "domain": "testok.baidu.com",
        "path": "/etc/nginx/cmdb/testok_baidu_com.conf",
        "created_at": "2026-06-12 10:00:00"
      }
    ],
    "total": 1
  }
}
```

### 下拉选项

返回可选的主域名列表(来自 `config.yaml` 中 `domains` 配置)。

```
GET /api/nginx/options
```

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "domains": [
      { "name": "hzlsg.com" },
      { "name": "xdysh.com" }
    ]
  }
}
```

### 健康检查

```
GET /health
```

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "ok",
    "template_ready": true
  }
}
```

> `status` 取值:
> - `ok`: 服务正常,模板就绪
> - `degraded`: 模板未就绪,接口不可用

## 项目结构

```
nginx-plugs/
├── main.go                          # 程序入口
├── go.mod
├── config/
│   ├── config.go                    # 配置加载
│   └── config.yaml                  # 配置文件
├── model/
│   ├── request.go                   # 请求模型
│   └── response.go                  # 响应模型
├── router/
│   └── router.go                    # 路由管理
├── api/
│   └── generate.go                  # 接口处理(添加/预览/删除/列表/选项)
└── common/
    ├── logging.go                   # 日志
    ├── response.go                  # 统一响应
    ├── template.go                  # 模板渲染与文件操作
    ├── nginx.go                     # SSH 重载 nginx
    └── dns.go                       # 阿里云 DNS 管理
```
