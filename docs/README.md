# 文档清单

本文档是 Task Platform 的文档入口，先列出项目需要维护的文档清单，再给出建议阅读顺序。项目已有的项目介绍、Postman 集合和面试讲解稿继续保留；本次补齐的是开发、交付和运维会实际用到的分册。

## 文档总览

| 文档 | 路径 | 读者 | 用途 |
|------|------|------|------|
| 项目介绍 | [project-introduction.md](project-introduction.md) | 面试官、评审、项目新成员 | 快速理解项目定位、业务能力和技术亮点 |
| Go 学习路线 | [../study.md](../study.md) | Go 初学者、新成员 | 按项目代码学习 Go 语法、分层、并发、测试和工程化实践 |
| 架构设计 | [architecture.md](architecture.md) | 后端开发、架构评审 | 说明服务边界、调用链路、认证契约和关键设计决策 |
| API 参考 | [api-reference.md](api-reference.md) | 前端开发、测试、接口调用方 | 汇总 HTTP 接口、请求参数、统一响应、错误码和幂等规则 |
| 配置说明 | [configuration.md](configuration.md) | 开发、运维 | 说明配置文件、环境变量、端口和 secret 要求 |
| 数据库设计 | [database-design.md](database-design.md) | 后端开发、DBA | 说明 schema、表结构、索引、软删除和迁移策略 |
| 开发指南 | [development-guide.md](development-guide.md) | 后端开发、新成员 | 说明本地环境、启动流程、代码边界和常见开发任务 |
| 测试指南 | [testing-guide.md](testing-guide.md) | 开发、测试、CI 维护者 | 说明单元、集成、前端、E2E、覆盖率和压测方式 |
| 部署指南 | [deployment-guide.md](deployment-guide.md) | 运维、开发 | 说明 Docker Compose 栈、本地部署顺序、健康检查和生产注意事项 |
| 可观测性运维 | [observability.md](observability.md) | 运维、后端开发 | 说明日志、指标、Grafana、Prometheus、Jaeger 和排障入口 |
| 前端指南 | [frontend-guide.md](frontend-guide.md) | 前端开发、全栈开发 | 说明 Web 应用结构、开发模式、Mock、构建和测试 |
| Postman 集合 | [postman/task-platform.postman_collection.json](postman/task-platform.postman_collection.json) | 接口调试者 | 提供可导入的接口集合和示例请求 |
| 面试讲解稿 | [interview-script.md](interview-script.md) | 项目作者、面试准备 | 提供项目讲解脉络、亮点和常见追问回答 |

## 建议阅读顺序

1. 快速了解项目：先读 [project-introduction.md](project-introduction.md)，再读 [architecture.md](architecture.md)。
2. 本地跑起来：按 [development-guide.md](development-guide.md) 准备依赖、启动基础设施并运行三个 Go 服务。
3. 对接接口：读 [api-reference.md](api-reference.md)，或直接导入 Postman 集合调试。
4. 修改后端能力：结合 [database-design.md](database-design.md)、[configuration.md](configuration.md) 和 proto 文件确认边界。
5. 验证质量：按 [testing-guide.md](testing-guide.md) 运行后端、前端、E2E 和压测。
6. 部署和排障：上线前读 [deployment-guide.md](deployment-guide.md)，运行中读 [observability.md](observability.md)。
7. 系统学习 Go：Go 初学者按 [../study.md](../study.md) 的 Phase 顺序读源码和做练习。

## 文档维护规则

- API 路由以 `internal/gateway/server/server.go` 为准。
- RPC 契约以 `api/proto/user/v1/user.proto` 和 `api/proto/task/v1/task.proto` 为准。
- 数据库结构以 `migrations/*/*.up.sql` 为准。
- 启动和质量命令以 `Makefile`、`web/package.json` 为准。
- 配置默认值以 `configs/<env>/*.yaml` 和 `.env.example` 为准。
