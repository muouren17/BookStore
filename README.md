# BookStore

一个使用 Go 编写的书店后端练习项目，提供书架与图书的基础管理接口。

这个项目是一次尝试把代码写得更规范的实践：以较小的业务场景，练习目录组织、职责分层、接口抽象、错误处理以及数据库事务。当前实现集中在书架和图书管理。

## 技术栈

- **Go**：项目语言，`go.mod` 声明版本为 `1.26.8`。
- **gRPC / Protocol Buffers**：定义服务接口与消息，并生成通信代码。
- **GORM / MySQL**：数据访问与持久化。
- **Viper / fsnotify**：读取 YAML 配置并监听配置文件变化。

## 已实现功能

- 创建、查询、列出和删除书架。
- 在指定书架中创建、查询、列出和删除图书。
- 创建书架时校验主题，设置默认容量。
- 添加图书时检查书架容量，在事务中锁定书架记录后统计数量、写入图书。
- 删除书架时，在同一事务中删除该书架及其图书。
- 将部分业务错误转换为 `InvalidArgument`、`NotFound`、`Internal` 等 gRPC 状态码。

部分业务规则：

- 书架主题去除首尾空白后不能为空。
- 书架容量的有效范围为 `1–5000`；未提供或超出范围时使用默认值 `500`。
- 图书作者为空或仅含空白时使用 `anonymous`，标题为空或仅含空白时使用 `others`。
- 图书属于指定书架，查询和删除图书时需要同时提供书架 ID 与图书 ID。

## 代码组织

```text
.
├── main.go                    # 初始化依赖、组装各层并启动 gRPC 服务
├── configs/
│   └── config.yaml            # 本地配置，已被 .gitignore 忽略，克隆后需自行创建
├── settings/
│   └── setting.go             # 配置结构与配置加载
├── internal/
│   ├── handler/               # 请求校验、消息转换与 gRPC 错误映射
│   ├── service/               # 业务规则与 Repository 接口定义
│   ├── dao/mysql/             # 基于 GORM 的 Repository 实现、事务与数据库初始化
│   └── model/                 # 数据模型
├── pb/
│   ├── bookstore.proto        # 服务与消息定义
│   ├── bookstore.pb.go        # 生成的消息代码
│   ├── bookstore_grpc.pb.go   # 生成的 gRPC 服务代码
│   └── bookstore.pb.gw.go     # 生成的 HTTP 网关代码，当前尚未接入启动流程
├── go.mod
└── go.sum
```

主要调用关系：

```text
客户端 → gRPC Handler → Service → Repository 接口 → MySQL 实现 → 数据库
```

`main.go` 负责创建对象并注入依赖；Service 通过 Repository 接口访问数据，由 MySQL 实现完成具体读写。

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

程序优先查找 `./configs/config.yaml`，也支持在当前目录查找 `config.yaml`。配置文件变化时会重新读取配置对象，但数据库连接池和监听端口不会随之重建，修改这些配置后需重启服务。

### 4. 安装依赖并启动

```bash
go mod download
go run .
```

默认监听 `8090` 端口，启动成功后会输出类似日志：

```text
gRPC listening [::]:8090
```

已生成的 Go 通信代码包含在仓库中，直接运行项目无需重新生成 Proto 代码。

## 接口概览

服务名称为 `pb.Bookstore`，接口定义见 [`pb/bookstore.proto`](pb/bookstore.proto)。

| RPC | 用途 |
| --- | --- |
| `ListShelves` | 列出所有书架 |
| `CreateShelf` | 创建书架 |
| `GetShelf` | 查询指定书架 |
| `DeleteShelf` | 删除书架及其中的图书 |
| `ListBooks` | 列出指定书架中的图书 |
| `CreateBook` | 向指定书架添加图书 |
| `GetBook` | 查询指定书架中的指定图书 |
| `DeleteBook` | 删除指定书架中的指定图书 |

当前启动的是明文 gRPC 服务。Proto 中已定义 HTTP 路由注解，仓库也包含生成的网关代码，但 `main.go` 尚未启动 HTTP 网关，不能直接用 HTTP 请求访问 `/v1/shelves` 等路径。

### 调用示例

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
- 区分业务错误与通信层错误。
- 使用事务和行锁处理容量校验及关联删除。
- 用 Proto 维护接口定义。

当前仓库未包含自动化测试，尚未实现用户认证、分页、图书更新或 HTTP 网关启动。后续可围绕这些部分继续练习和完善。
