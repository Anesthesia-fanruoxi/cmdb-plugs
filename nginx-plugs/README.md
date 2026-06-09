# Nginx Plugs

Nginx 配置文件管理插件，通过模板自动生成、查看、删除 nginx 站点配置。

## 快速开始

```bash
# 下载依赖
go mod tidy

# 运行
go run main.go
```

服务默认运行在 `http://localhost:8091`

## 配置

配置加载优先级：环境变量 > 配置文件 > 默认值

### 配置文件

编辑 `config/config.yml`：

```yaml
server:
  port: 8091

nginx:
  conf_dir: "/etc/nginx/conf.d/cmdb-plugs/"   # 配置文件输出目录
  template_file: "templates/proxy.conf.template"  # 模板文件路径

log:
  level: "info"
```

### 环境变量

```bash
NGINX_PLUGS_PORT=8091
NGINX_CONF_DIR=/etc/nginx/conf.d/cmdb-plugs/
NGINX_TEMPLATE_FILE=templates/proxy.conf.template
LOG_LEVEL=info
```

## nginx 主配置

在 nginx 主配置文件中添加一行 include，加载程序生成的配置：

```nginx
include /etc/nginx/conf.d/cmdb-plugs/*.conf;
```

这样程序只在 `cmdb-plugs/` 子目录内操作，不会影响其他 nginx 配置。

## API 接口

### 生成配置

根据模板生成 nginx 配置文件并写入配置目录（覆盖写入）。

```
GET /api/nginx/generate?server_name=api.example.com
```

参数：
- `server_name`（必填）：域名

### 预览配置

读取已生成的配置文件内容。

```
GET /api/nginx/preview?server_name=api.example.com
```

参数：
- `server_name`（必填）：域名

### 删除配置

删除指定域名的配置文件。

```
DELETE /api/nginx/delete?server_name=api.example.com
```

参数：
- `server_name`（必填）：域名

### 配置列表

列出配置目录中所有已生成的 `.conf` 文件。

```
GET /api/nginx/list
```

### 健康检查

```
GET /health
```

## 项目结构

```
nginx-plugs/
├── main.go                          # 程序入口
├── go.mod
├── config/
│   ├── config.go                    # 配置加载
│   └── config.yml                   # 配置文件
├── templates/
│   └── proxy.conf.template          # nginx 配置模板
├── model/
│   ├── request.go                   # 请求模型
│   └── response.go                  # 响应模型
├── router/
│   └── router.go                    # 路由管理
├── api/
│   └── generate.go                  # 接口处理
└── common/
    ├── logging.go                   # 日志
    ├── response.go                  # 统一响应
    └── template.go                  # 模板渲染与文件操作
```
