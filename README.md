# multipackage-release-planner

离线多包版本发布规划工具。读取 JSON 输入，根据约定式提交（Conventional
Commits）计算每个包的新版本并生成变更记录。**只生成计划**：不修改仓库，
不调用任何发布服务。仅使用 Go 标准库。

## 用法

```sh
go run ./cmd/release-planner [-markdown plan.md] input.json
```

JSON 计划输出到标准输出；加 `-markdown` 可同时写出 Markdown 变更记录。
示例输入见 `examples/input.json`。

## 输入格式

```json
{
  "packages": [{ "name": "web", "version": "1.9.9" }],
  "commits": [
    { "id": "c1", "message": "fix: shared patch", "packages": ["web"] }
  ]
}
```

- 版本必须是三段非负整数（`major.minor.patch`），不允许前导零。
- 每条提交有唯一 `id`、原始 `message` 和明确的影响包集合 `packages`。
- 未知包、重复提交 id、非法版本等任何问题都会拒绝整批输入。

## 提交消息格式

首行：

```
type[(scope)][!]: description
```

- `type` 为小写英文字母（如 `feat`、`fix`、`chore`、`refactor`）。
- `scope` 可选，仅作记录，不改变显式的影响包集合。
- 冒号后必须有一个空格，且说明非空；格式错误会定位到具体提交 id。

破坏性变更（满足任一）：

- 首行在冒号前带 `!`，如 `feat!: drop v1`；
- 正文独立一行以 `BREAKING CHANGE: ` 开头且说明非空。

正文中普通提及该词不算；任何类型（如 `perf`、`refactor`）都可以是破坏性变更。

## 版本规则

- 破坏性变更提升主版本，`feat` 提升次版本，`fix` 提升修订版本，其他类型不提版本。
- 每个包只采用本批最高级别，低位归零（如 `1.9.9` minor → `1.10.0`）。
- 零主版本同样遵循该规则（`0.2.3` major → `1.0.0`）。
- 没有变更的包保持原版本。

## 输出

- JSON 计划：包按名称排序，每个包给出当前版本、新版本、提升级别和
 佐证提交（按级别 major → minor → patch 分组，组内保持原提交顺序，
 同一提交不重复出现）。
- Markdown 变更记录：按包分节，组标题为 Breaking Changes / Features /
 Bug Fixes。

## 测试

```sh
go test ./...
```
