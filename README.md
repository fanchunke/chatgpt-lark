
# chatgpt-lark

飞书接入 GPT3 接口。可以创建飞书应用，体验 ChatGPT。

相较于官方提供的 `CreateCompletion` 接口，该项目增加了会话管理功能，能够较好地提供多轮对话能力。

## 快速开始

1. 修改配置

修改 `conf/online.conf` 文件，主要涉及飞书应用配置、GPT3 API Key、会话管理数据库配置等。

- 飞书应用配置
    - 参考 [自建应用的开发流程](https://open.feishu.cn/document/home/introduction-to-custom-app-development/self-built-application-development-process)
- Open AI Key
  - 需要自行申请
- 数据库
  - ~数据库需要自行创建，数据表的创建可以通过命令行方式执行。~
  - 数据库支持 sqlite3，可以通过修改配置使用。如果使用 MySQL，需要自行创建数据库。
  - **数据表在程序启动时自动创建。**

2. `Docker` 运行

```shell
docker-compose up -d
```

3. 初始化数据表

数据表在程序启动时自动创建。

4. 配置飞书应用
    - 在飞书应用配置后台，配置【事件订阅】-【请求地址配置】，格式：`http[s]://ip:port/lark/receive`

## FAQ

**怎么创建数据库**

- v0.1.1 版本中支持 sqlite3 数据库，只需要修改配置文件的配置，程序启动后便会初始化数据库和数据表，不需要额外的操作。

- 如果使用的是 MySQL，则需要自行创建数据库，建库 SQL 可以参考下面的命令：

```sql
CREATE DATABASE chatgpt DEFAULT CHARACTER SET utf8mb4
```

之后程序启动后，便可以自动创建数据表。

**数据库连接失败**

- 首先检查数据库配置是否正确
- 如果使用 docker 部署服务，需要确认容器内能否连接到数据库。最常见的一个问题是，在宿主机部署了 MySQL，但是在容器内配置 `127.0.0.1`，这种情况需要配置宿主机的 IP

**数据库配置说明**

v0.1.1 版本可以支持 MySQL、SQLite、PostgreSQL。常见的配置如下：

MySQL:

```toml
[database]
# mysql
driver="mysql"
dataSource="root:12345678@tcp(127.0.0.1:3306)/chatgpt?parseTime=True"
```

SQLite

```toml
[database]
# sqlite3
driver="sqlite3"
dataSource="file:chatgpt?_fk=1&parseTime=True"
```

## Changelog

### v0.1.1

- 修复 prompt 过长导致接口调用失败问题
- 支持 sqlite3
- 自动初始化数据库
- 支持关闭会话功能

### v0.1.0

- 项目初始化