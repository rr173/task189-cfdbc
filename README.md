# task189-cfdbc — 计算流体网格边界条件一致性服务

基于 REQ-20260823-052 生成。流体仿真工程师登记网格区域与面摘要、物理模型与边界条件；服务校验每个外露面恰有一种适用条件、耦合面两侧守恒、入口/出口质量流量总量平衡、不可压缩模型参考压力可解性，输出问题清单；通过校验的配置发布为求解前置包，修改条件或网格后派生新包。

## 业务闭环

1. 登记网格区域（维度/单元规模/描述），导入面集合（外露/耦合/退化/重复分类）。
2. 连通性检测：耦合面必须有对侧区域与对侧面；区域状态推进 connected / isolated。
3. 创建并激活物理模型（可压缩/不可压缩 → 是否强制参考压力）。
4. 分配边界条件（入口/出口质量流量、压力出口、壁面、对称、耦合面 A/B 侧、参考压力），同面重复条件立即标记冲突。
5. 一致性校验：覆盖检查（外露面恰一条件）、耦合面双侧守恒（单位一致 + 规格一致）、入口/出口质量流量总量平衡、参考压力可解性；输出问题清单与结论（solvable / underconstrained / overconstrained）。
6. 通过校验后发布求解前置包（绑定区域拓扑哈希 + 条件版本 + 不可变快照）；已发布包不可直接改写，修改配置派生新包。
7. 相同网格哈希幂等导入拒绝；重启后运行记录、包状态、统计全部恢复。

## 状态机

- 网格区域：`imported → connected / isolated → sealed`（sealed 终态不可逆）
- 面：`unclassified → exposed / coupled / duplicate / degenerate`
- 边界条件：`draft → applicable / conflicting / missing → approved`（乐观锁版本递增）
- 求解前置包：`building → solvable / underconstrained / overconstrained → published`（published 不可改写）

## 标准命令

```bash
# 构建
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...

# 静态检查
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...

# 单元测试
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...

# 端到端自检（关闭重开数据库验证重启恢复）
go run ./cmd/cfdbc --smoke-test

# 启动服务
go run ./cmd/cfdbc --addr :8080 --db cfdbc.db
```

## API 一览（前缀 /api）

| 能力 | 入口 |
|---|---|
| 健康/统计 | `GET /api/health` `GET /api/stats` |
| 区域登记/列表/读取/封存 | `POST /api/regions` `GET /api/regions` `GET /api/regions/{id}` `POST /api/regions/{id}/seal` |
| 面导入/列表 | `POST /api/regions/{id}/faces` `GET /api/regions/{id}/faces` |
| 面详情/面条件 | `GET /api/faces/{id}` `GET /api/faces/{id}/conditions` |
| 物理模型创建/列表/读取/激活 | `POST /api/models` `GET /api/models` `GET /api/models/{id}` `POST /api/models/{id}/activate` `GET /api/models/active` |
| 条件分配/列表/读取/批准/修订 | `POST /api/conditions` `GET /api/conditions` `GET /api/conditions/{id}` `POST /api/conditions/{id}/approve` `POST /api/conditions/{id}/revise` |
| 一致性校验/运行记录 | `POST /api/validate` `GET /api/runs` `GET /api/runs/{id}` `GET /api/runs/{id}/issues` |
| 问题清单/详情 | `GET /api/issues` `GET /api/issues/{id}` |
| 前置包构建/列表/读取/发布/派生/差异 | `POST /api/packages` `GET /api/packages` `GET /api/packages/{id}` `POST /api/packages/{id}/publish` `POST /api/packages/derive` `GET /api/packages/{id}/diff?against=` |

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。表：`regions`、`faces`、`physics_models`、`boundary_conditions`、`validation_runs`、`validation_issues`、`solver_packages`。重启后运行记录与包状态恢复；相同网格哈希幂等导入拒绝；发布包冻结区域拓扑哈希与条件版本快照。
