# Gooodclass Backend

安徽理工大学「好好好课程表」后端服务

## 技术栈

- **Go** + [Gin](https://github.com/gin-gonic/gin) Web 框架
- **MySQL** 数据存储
- **SQLite** 签到数据本地持久化
- **Docker** 容器化部署

## 功能模块

| 模块 | 说明 |
|------|------|
| `login` | 教务系统登录认证 |
| `schedule` | 课程表查询 |
| `exam` | 考试安排查询 |
| `gpa` | 成绩与绩点查询 |
| `rank` | 排名查询 |
| `emptyroom` | 空教室查询 |
| `checkin` | 签到功能 |
| `campus` | 校区信息 |
| `plan` | 教学计划 |
| `review` | 课程评价 |
| `vote` | 投票功能 |
| `qa` | 问答功能 |
| `unscheduled` | 非排课活动 |

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Docker & Docker Compose（可选）

### 本地运行

```bash
# 安装依赖
go mod download

# 设置环境变量
export DATA_DIR=./data
export MYSQL_DSN="user:password@tcp(localhost:3306)/gooodclass?charset=utf8mb4&parseTime=True"

# 运行
go run .
```

### Docker 部署

```bash
docker-compose up -d
```

## 项目结构

```
├── main.go                 # 入口文件，路由注册
├── internal/
│   ├── auth/               # 认证客户端
│   ├── handler/            # 请求处理器
│   ├── jwgl/               # 教务系统客户端
│   ├── middleware/          # 中间件
│   └── store/              # 数据存储层
├── data/                   # 数据文件
├── Dockerfile
└── docker-compose.yml
```

## License

MIT
