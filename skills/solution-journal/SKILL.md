---
name: solution-journal
description: >-
  边解题边沉淀「解题思路」与「踩坑」：用 note/ Fact 记录推理路径、假设、关键决策，
  以及已排除的死路、陷阱、失败根因与教训，服务决策可解释、行为可复现、上下文压缩后不丢上下文。
  Use when recording the reasoning/approach behind a decision, or a pitfall/dead-end/lesson
  encountered while solving a task.
metadata:
  tags: [解题思路, 踩坑, 决策可解释, 行为可复现, 知识沉淀, 复盘]
---

# 解题思路与踩坑日志（solution-journal）

> 状态落在**已绑定项目**的 SQLite 黑板（与 pentest-blackboard 同库），用 `note/` 类 Fact 记录「为什么这么做」与「踩过的坑」。
> **分工**：pentest-blackboard 记「是什么」（目标 / 认证 / 发现 / 漏洞），本 skill 记「**为什么 / 怎么绕**」（解题思路 + 踩坑），两者可各记一次、用 links 串联。
> 系统只注入黑板索引（`fact_key` + summary）；**要拿完整思路/坑细节必须 `get_project_fact`，禁止凭摘要臆造。**

## 两类记录

| 记录 | fact_key | confidence | 触发时机 |
|------|----------|------------|----------|
| **解题思路** | `note/approach-<slug>` | 执行中 `tentative`；已验证有效 `confirmed` | 选路径 / 下决策 / 定假设 / 定计划时 |
| **踩坑** | `note/pitfall-<slug>` | 已证实 `confirmed`；仅怀疑 `tentative` | 失败 / 死路 / 报错 / 绕弯路 / 工具坑时 |

- **slug 要自解释**：`note/approach-sqli-time-based`、`note/pitfall-waf-403-rate-limit`，一眼看出是思路还是坑。
- **同一件事保持同一 `fact_key` 覆盖更新**，勿拆成多个 key（思路演进改同一条，坑被绕过后补「解法」字段）。
- 被否定的思路：不删，改 `confidence=tentative` 并在 body「为什么放弃」里写明，或直接并入对应 `pitfall`。

## 解题思路怎么记（note/approach-*）

**summary**（索引用一行）＝「面对什么问题 + 选了哪条路 + 依据是什么」。禁止只写「试试注入」。

**body** 建议骨架：

```markdown
## 目标 / 问题
<当前要解决的具体子问题，不是大目标>

## 假设
<触发条件、预期行为、为什么这么想>

## 候选路径与决策
| 路径 | 依据 | 取舍 |
|------|------|------|
| A   | ...  | 选它 / 否掉（理由）|

- 已选: <路径 + 为什么>
- 已排除: <路径 + 为什么（证据：工具输出、响应、文档）>

## 下一步
<可执行的下一步，而非泛泛描述>

## 关联
- 依据 fact_key: <如 target/*、finding/*>
- links: [{from: <相关fact>, type: leads_to|depends_on|supports}]
```

## 踩坑怎么记（note/pitfall-*）

**summary**（一行）＝「什么现象 + 在哪触发 + 怎么绕开」。禁止只写「遇到个坑」。

**body** 建议骨架：

```markdown
## 现象
<报错 / 死路 / 被拦 / 结果异常，贴原始输出片段>

## 触发条件
<什么操作 / 什么输入 / 什么环境触发>

## 根因
<为什么：框架限制、目标防御、工具缺陷、自己误判>

## 尝试与排除
<已试的绕过（按时间），各自结果；避免别人重走>

## 解法 / 规避
<最终怎么绕过去，或为什么放弃换路>

## 教训（可复用）
<一条能迁移到其它题的判断/操作>

## 关联
- 相关思路 fact_key: <如 note/approach-*>
- links: [{from: <相关fact>, type: supports|part_of}]
```

## 强制节奏：边解边记

勿等会话结束再批量补写，否则上下文压缩后思路/坑就丢了。

1. 每**下决策 / 换路径 / 定假设** → 立即 `upsert_project_fact` 写 `note/approach-*`（同 key 覆盖）。
2. 每**失败 / 死路 / 报错 / 踩坑** → 立即写 `note/pitfall-*`，根因没查清也先 `tentative` 落库，查清再补根因。
3. 继续下一步前优先落库，避免压缩丢细节。
4. **未绑项目**：说明无法写黑板，仍在本轮回复末尾给「待落库」条目（`fact_key` + summary + body/POC 要点），供协调者写入。
5. **协调者**：子任务返回的思路/坑由协调者汇总写入，勿假定子代理已记。
6. **子代理无工具时**：交付物末尾给「待落库」思路/坑条目，供协调者立即写入。

## 工具速查

| 工具 | 用途 |
|------|------|
| `upsert_project_fact` | 写入/更新思路与坑（category=`note`，含 summary/body/confidence/links） |
| `get_project_fact` | 按 key 取完整 body（索引不够时必调） |
| `list_project_facts` / `search_project_facts` | 检索已记的思路与坑（避免重走死路） |
| `deprecate_project_fact` / `restore_project_fact` | 思路/坑已失效则废弃 / 恢复 |

前置：**当前对话已绑定项目**（否则工具报错）。

## 关系边 links（串联「是什么」与「为什么」）

- 思路依赖事实：`{from: target/*, type: depends_on}`；思路引出发现：`{from: note/approach-*, type: leads_to}`（指向 `finding/*`）。
- 坑支撑思路：`{from: note/pitfall-*, type: supports}`；坑属于某发现：`{from: note/pitfall-*, type: part_of}`（指向 `finding/*`）。
- 常用 type：`leads_to` \| `depends_on` \| `enables` \| `supports` \| `part_of` \| `contains`。
- **省略 links** = 保留已有边；传入 links = **替换**全部入边。

## 行为触发器（纪律）

每步必检（不检 = 违反验证铁律）：

1. **下决策即记思路**：换了攻击路径 / 改了假设 / 定了计划 → 先 `upsert_project_fact(note/approach-*)` 再执行，让「为什么这么做」可审计。
2. **踩坑即记坑**：报错 / 被拦 / 死路 / 工具异常 → 先 `upsert_project_fact(note/pitfall-*)` 再换路，避免重走。
3. **先查再走**：开新路径前 `search_project_facts` 扫一遍已有 `note/pitfall-*` 与 `note/approach-*`，有可复用的坑/思路就不要从零试。
4. **负结果也是坑**：验证失败、绕过失败、环境差异，都要落 `note/pitfall-*`（`confirmed` + body 写清已测条件），不落 = 让后人重踩。

违反判定：下决策不记思路 = 不可解释；踩坑不记坑 = 不可复现；重走已记的死路 = 没先查黑板 → 均违反验证铁律。

## 与其它 skill 的衔接

- **pentest-blackboard**：本 skill 只记 `note/approach-*`、`note/pitfall-*`；事实（target/auth/infra/business）与发现（finding/chain/exploit/poc）仍走 pentest-blackboard，用 links 串联。
- **pentest-verification**：搜索 ≠ 漏洞；无证据不写 `confirmed`；负结果落 `note/pitfall-*` 而非 `finding/*`。
