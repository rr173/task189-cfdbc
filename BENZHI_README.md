# task189-cfdbc 评测说明

## 项目简介

计算流体网格边界条件一致性服务：登记 CFD 网格区域与面、物理模型与边界条件，校验覆盖性、耦合面守恒、质量流量平衡与参考压力可解性，输出问题清单；通过后发布求解前置包，修改派生新包。

## 标准命令

```bash
# 构建 / 静态检查 / 单元测试
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...

# 端到端自检（唯一判据）：真实走完登记→导入→模型→条件→校验→发布→派生→重启恢复
go run ./cmd/cfdbc --smoke-test
```

## 服务入口

```bash
go run ./cmd/cfdbc --addr :8080 --db cfdbc.db
```

- `--smoke-test`：不驻留，执行端到端闭环并断言关键不变量（耦合面单侧欠约束 → 补齐 → 冲突检测 → 参考压力 → 可求解 → 发布 → 派生 → 重开数据库恢复），退出码 0 为通过。
- HTTP API 前缀 `/api`（见 README 表）。

## Docker 双架构

```bash
# 构建镜像（默认 linux/amd64）
bash build_benzhi_docker.sh <镜像名> linux/amd64
bash build_benzhi_docker.sh <镜像名> linux/arm64

# 运行自检（Dockerfile ENTRYPOINT+CMD 已设，只传 flag）
docker run --rm <镜像名> --smoke-test

# 启动服务
docker run --rm -p 8080:8080 <镜像名> --addr :8080
```

## --smoke-test 契约

- 创建两个区域、导入含耦合面与出入口的面集合；
- 激活不可压缩模型（强制参考压力）；
- 分配条件并断言：耦合面单侧 → `underconstrained`；同面双条件 → 新条件 `conflicting`；缺参考压力 → 仍欠约束；补齐耦合面 B 侧与参考压力 → `solvable`；
- 发布包 → `published`；派生新包 → ID 不同、diff 可计算；
- 关闭并重新打开同一数据库：运行结果、包状态、统计一致；相同网格哈希重复导入被拒绝。
