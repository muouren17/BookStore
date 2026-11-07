# BookStore

一个使用 Go 编写的书店后端练习项目，通过 gRPC 和 HTTP/JSON 提供书架与图书的基础管理接口。

这个项目是一次尝试把代码写得更规范的实践：以较小的业务场景，练习目录组织、职责分层、接口抽象、错误处理以及数据库事务。当前实现集中在书架和图书管理。

## 技术栈

- **Go**：项目语言，`go.mod` 声明版本为 `1.26.8`。
- **gRPC / Protocol Buffers**：定义服务接口与消息，并生成通信代码。
- **gRPC-Gateway**：将 HTTP/JSON 请求转发到本机的 gRPC 服务。
- **GORM / MySQL**：数据访问与持久化。
- **Viper / fsnotify**：读取 YAML 配置并监听配置文件变化。

## 已实现功能

- 创建、查询、列出和删除书架。
- 在指定书架中创建、查询、列出和删除图书。
- 创建书架时校验主题，设置默认容量。
- 添加图书时检查书架容量，在事务中锁定书架记录后统计数量、写入图书。
- 删除书架时，在同一事务中删除该书架及其图书。
- 将部分业务错误转换为 `InvalidArgument`、`NotFound`、`Internal` 等 gRPC 状态码。
- 同时启动 gRPC 服务与 HTTP Gateway，使用不同的监听端口。
- 启动时校验 MySQL 配置是否存在、各端口是否有效，以及 gRPC 与 HTTP 端口是否冲突。
- 收到 Ctrl+C 或 `SIGTERM` 时关闭服务，共用 10 秒的退出期限；超时后强制停止未结束的服务。

部分业务规则：

- 书架主题去除首尾空白后不能为空。
- 书架容量的有效范围为 `1–5000`；未提供或超出范围时使用默认值 `500`。
- 图书作者为空或仅含空白时使用 `anonymous`，标题为空或仅含空白时使用 `others`。
- 图书属于指定书架，查询和删除图书时需要同时提供书架 ID 与图书 ID。

## 代码组织

```text
.
├── main.go                    # 初始化依赖、启动 gRPC 与 HTTP Gateway、处理退出信号
├── configs/
│   └── config.yaml            # 本地配置，已被 .gitignore 忽略，克隆后需自行创建
├── settings/
│   ├── setting.go             # 配置结构、加载与启动校验
│   └── setting_test.go        # 配置校验及缺少 MySQL 配置的测试
├── internal/
│   ├── handler/               # 请求校验、消息转换与 gRPC 错误映射
│   ├── service/               # 业务规则与 Repository 接口定义
│   ├── dao/mysql/             # 基于 GORM 的 Repository 实现、事务与数据库初始化
│   └── model/                 # 数据模型与共享错误定义（ErrNotFound、ErrShelfFull）
├── pb/
│   ├── bookstore.proto        # 服务与消息定义
│   ├── bookstore.pb.go        # 生成的消息代码
│   ├── bookstore_grpc.pb.go   # 生成的 gRPC 服务代码
│   └── bookstore.pb.gw.go     # 生成的 HTTP 网关代码，已接入启动流程
├── go.mod
└── go.sum
```

主要调用关系：

```text
客户端 → gRPC Handler → Service → Repository 接口 → MySQL 实现 → 数据库
HTTP 客户端 → HTTP Gateway → 本机 gRPC 服务 → 上述调用链
```

`main.go` 负责创建对象并注入依赖；Service 通过 Repository 接口访问数据，由 MySQL 实现完成具体读写。MySQL 层将记录不存在等错误转换为 model 层的共享错误，Service 据此判断业务结果，无需直接依赖 GORM 或 MySQL 实现。

## 本地运行

### 1. 准备环境

- 安装符合 `go.mod` 要求的 Go 工具链。
- 准备可连接的 MySQL 实例，使用支持事务和行锁的 InnoDB 表。
- 在项目根目录执行后续 Go 命令。

### 2. 准备数据库表

项目包含 `internal/dao/mysql/migrate.go` 中的 `CreateTable` 函数，但当前启动流程没有调用它。首次运行需先准备数据库和表。

以下 SQL 可用于本地练习环境；数据库名可自行调整，并与配置保持一致：

```sql
CREATE DATABASE IF NOT EXISTS bookstore CHARACTER SET utf8mb4;
USE bookstore;

CREATE TABLE IF NOT EXISTS shelves (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    theme LONGTEXT,
    size BIGINT,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS books (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    author LONGTEXT,
    title LONGTEXT,
    shelf_id BIGINT NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    INDEX idx_books_shelf_id (shelf_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3. 配置连接信息

创建或修改 `configs/config.yaml`：

```yaml
name: "bs"
port: 8090
http_port: 8091

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "your_mysql_user"
  password: "your_mysql_password"
  dbname: "bookstore"
  max_open_conns: 200
  max_idle_conns: 50
```

将示例中的连接信息替换为自己的配置。`configs/config.yaml` 已被 Git 忽略，避免把本地数据库凭据提交到仓库。

`port` 是 gRPC 端口，`http_port` 是 HTTP Gateway 端口，两项都必须填写且不能相同。启动校验要求这些端口及 `mysql.port` 均在 `1–65535` 范围内，并且必须提供 `mysql` 配置段；程序没有为这两个监听端口设置默认值。旧配置需要补上 `http_port`。

程序优先查找 `./configs/config.yaml`，也支持在当前目录查找 `config.yaml`。当前 `.gitignore` 只排除了前者，建议将本地配置放在 `configs/` 下。配置文件变化时会重新读取配置对象，但该回调不会再次执行启动校验，数据库连接池和监听端口也不会随之重建，修改这些配置后需重启服务。

### 4. 安装依赖并启动

```bash
go mod download
go run .
```

使用上述示例配置时，gRPC 监听 `8090` 端口，HTTP Gateway 监听 `8091` 端口，程序会输出：

```text
gRPC 端口: 8090，HTTP Gateway 端口: 8091
```

已生成的 Go 通信代码包含在仓库中，直接运行项目无需重新生成 Proto 代码。

按 Ctrl+C 或发送 `SIGTERM` 可触发服务关闭：先等待 HTTP 请求结束，再等待 gRPC 请求结束，两个阶段共用 10 秒期限，最后关闭数据库连接池。

### 5. 运行测试

```bash
go test ./...
```

当前自动化测试位于 `settings/setting_test.go`，覆盖端口范围、监听端口冲突、缺少 MySQL 配置，以及配置加载时拒绝缺少 MySQL 配置的情况。这些测试不需要连接数据库；HTTP/gRPC 接口、业务逻辑和退出流程尚未包含自动化测试。

## 接口概览

服务名称为 `pb.Bookstore`，接口定义见 [`pb/bookstore.proto`](pb/bookstore.proto)。

| RPC | HTTP 方法与路径 | 用途 |
| --- | --- | --- |
| `ListShelves` | `GET /v1/shelves` | 列出所有书架 |
| `CreateShelf` | `POST /v1/shelves` | 创建书架 |
| `GetShelf` | `GET /v1/shelves/{id}` | 查询指定书架 |
| `DeleteShelf` | `DELETE /v1/shelves/{id}` | 删除书架及其中的图书 |
| `ListBooks` | `GET /v1/shelves/{id}/books` | 列出指定书架中的图书 |
| `CreateBook` | `POST /v1/shelves/{id}/books` | 向指定书架添加图书 |
| `GetBook` | `GET /v1/shelves/{shelfid}/books/{bookid}` | 查询指定图书 |
| `DeleteBook` | `DELETE /v1/shelves/{shelfid}/books/{bookid}` | 删除指定图书 |

当前 gRPC 与 HTTP 服务均使用明文传输，Gateway 通过 `127.0.0.1:<port>` 连接 gRPC 服务。

### HTTP 调用示例

以下命令使用示例 HTTP 端口 `8091`：

```bash
# 创建书架
curl -X POST http://localhost:8091/v1/shelves \
  -H 'Content-Type: application/json' \
  -d '{"theme":"Go 学习","size":"100"}'

# 列出所有书架
curl http://localhost:8091/v1/shelves

# 在书架中添加图书，将路径中的 1 替换为实际书架 ID
curl -X POST http://localhost:8091/v1/shelves/1/books \
  -H 'Content-Type: application/json' \
  -d '{"author":"示例作者","title":"Go 入门"}'

# 列出该书架上的图书
curl http://localhost:8091/v1/shelves/1/books

# 查询图书，将最后的 1 替换为实际图书 ID
curl http://localhost:8091/v1/shelves/1/books/1
```

HTTP 创建请求的 JSON 直接填写书架或图书字段，不需要再包一层 `shelf` 或 `book`；gRPC 请求仍使用 Proto 中定义的完整请求结构。Gateway 使用默认的 ProtoJSON 编解码方式，`int64` 类型的 ID、容量在响应 JSON 中表示为字符串。

### gRPC 调用示例

可以使用生成的 Go 客户端，或使用支持加载 Proto 文件的 gRPC 调试工具。

以下以 `grpcurl` 为例。服务当前未注册 gRPC Reflection，因此需要显式提供 Proto 文件；由于 Proto 引用了 `google/api/annotations.proto`，还需准备包含该文件的 Google APIs Proto 目录。

在项目根目录执行，把 `/path/to/googleapis` 替换为实际目录（该目录下应有 `google/api/annotations.proto` 及其依赖）：

```bash
# 创建书架
grpcurl -plaintext \
  -import-path . \
  -import-path /path/to/googleapis \
  -proto pb/bookstore.proto \
  -d '{"shelf":{"theme":"Go 学习","size":100}}' \
  localhost:8090 pb.Bookstore/CreateShelf

# 列出所有书架
grpcurl -plaintext \
  -import-path . \
  -import-path /path/to/googleapis \
  -proto pb/bookstore.proto \
  -d '{}' \
  localhost:8090 pb.Bookstore/ListShelves
```

## 实践重点与当前范围

这个练习主要关注：

- 按职责划分 Handler、Service 和数据访问层。
- 用接口描述数据访问能力，并通过构造函数注入依赖。
- 用共享错误定义隔离数据库错误、业务错误与通信层错误。
- 使用事务和行锁处理容量校验及关联删除。
- 用 Proto 同时维护 gRPC 接口定义与 HTTP 路由映射。
- 校验启动配置，并通过退出信号协调 HTTP 与 gRPC 服务的关闭。

当前已有配置相关的自动化测试，尚未实现用户认证、TLS、分页或图书更新，也尚未覆盖业务逻辑、接口和退出流程的自动化测试。后续可围绕这些部分继续练习和完善。
