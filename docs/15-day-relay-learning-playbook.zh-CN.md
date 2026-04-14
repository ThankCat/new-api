# 15天冲刺执行手册：从0实现AI中转服务（Go初学者版）

## 1. 使用方式（每天固定2小时）
1. `20分钟`：理解今天链路节点。
2. `70分钟`：只实现今天要求，不超范围。
3. `20分钟`：跑测试和接口验证。
4. `10分钟`：写复盘（今天学会了什么，哪里还不懂）。

每天都必须产出三样东西：
1. 代码提交或本地改动记录。
2. 可复现的测试命令和结果。
3. 复盘文档（建议写在 `learning-log/day-XX.md`）。

---

## 2. 一条总链路（15天不变）

`请求进入 -> 鉴权 -> 请求解析/校验 -> 模型选路 -> 协议转换 -> 上游调用 -> 流式回传 -> 用量统计 -> 计费结算 -> 日志审计 -> 管理接口`

你每天只推进这条链路上的一个节点，最后第15天整链打通。

## Day 1：请求进入与基础骨架

### 今天先读源码（阅读边界）
1. 从 `main.go` 的 `main()` 读到 `router.SetRouter(...)`。
2. 从 `router/main.go` 的 `SetRouter()` 读到 `SetApiRouter()`、`SetRelayRouter()`。
3. 从 `middleware/request-id.go` 的 `RequestId()` 读到 `middleware/logger.go` 的 `SetUpLogger()`。
阅读顺序：`main.go -> router/main.go -> router/api-router.go -> router/relay-router.go -> middleware/request-id.go`。
读后检查题：请求进入后，最先执行的中间件是哪一个，为什么。

### 今天搞清楚什么
1. Gin 服务启动流程。
2. 路由注册到处理函数的映射关系。
3. 中间件执行顺序。

### 今天实现什么
1. 新建你自己的最小中转项目骨架（建议单独目录，如 `my-gateway`）。
2. 实现 `GET /health`。
3. 实现 `RequestID` 中间件，给每个请求注入唯一ID。

### 今天怎么验证
1. `curl http://localhost:<port>/health` 返回200。
2. 连续请求两次，响应头或日志中的 `request_id` 不相同。

### 今天通过标准
1. 你能口述请求从 `main` 到 handler 的执行路径。
2. 你能解释中间件为什么先于 handler 执行。

---

## Day 2：Token鉴权链路

### 今天先读源码（阅读边界）
1. 从 `middleware/auth.go` 的 `TokenAuth()` 读到 `SetupContextForToken()`。
2. 从 `model/token.go` 的 `ValidateUserToken()` 读到 `GetTokenByKey()`。
3. 从 `router/api-router.go` 的 `/api/token` 路由组读到 `controller/token.go` 的 `AddToken()`、`UpdateToken()`、`GetTokenUsage()`。
阅读顺序：`router/api-router.go -> middleware/auth.go -> model/token.go -> controller/token.go`。
读后检查题：鉴权通过后，用户ID和token信息是怎么放进 `gin.Context` 的。

### 今天搞清楚什么
1. 什么是鉴权与授权。
2. API Token 在中转服务中的作用。
3. 鉴权失败应该如何返回（状态码和错误结构）。

### 今天实现什么
1. `Token` 数据结构（先内存版，后续接DB）。
2. `TokenAuth` 中间件。
3. 无Token、非法Token、合法Token三种分支处理。

### 今天怎么验证
1. 无 `Authorization` 请求必须被拒绝。
2. 错误Token必须被拒绝。
3. 合法Token能访问受保护接口。
4. 写至少1个鉴权单测。

### 今天通过标准
1. 你能解释为何鉴权放在中间件而不是业务handler里。
2. 你能清楚说出401和403的使用边界。

---

## Day 3：模型与渠道选路

### 今天先读源码（阅读边界）
1. 从 `middleware/distributor.go` 的 `Distribute()` 读到 `SetupContextForSelectedChannel()`。
2. 从 `service/channel_select.go` 的 `CacheGetRandomSatisfiedChannel()` 读到 `RetryParam` 相关方法。
3. 从 `model/channel_cache.go` 的 `GetRandomSatisfiedChannel()` 读到 `model/ability.go` 的 `GetChannel()`、`getPriority()`。
阅读顺序：`middleware/distributor.go -> service/channel_select.go -> model/channel_cache.go -> model/ability.go -> model/channel.go(GetNextEnabledKey)`。
读后检查题：为什么“按模型选路”不放在 controller，而放在 middleware + service。

### 今天搞清楚什么
1. `Model -> Channel` 映射关系。
2. 渠道状态、优先级、权重的作用。
3. 为什么要把选路逻辑独立成 service 层。

### 今天实现什么
1. `Channel`、`Ability` 结构体。
2. “按模型选可用渠道”的核心函数。
3. 过滤禁用渠道和空渠道场景。

### 今天怎么验证
1. 给定模型能选到预期渠道。
2. 渠道禁用后不会被选中。
3. 没有可用渠道时返回可读错误。

### 今天通过标准
1. 你能画出选路函数输入与输出。
2. 你能说明为何要提前判空和禁用状态。

---

## Day 4：DTO与零值保留

### 今天先读源码（阅读边界）
1. 从 `dto/openai_request.go` 的 `GeneralOpenAIRequest` 读到 `GetTokenCountMeta()`。
2. 从 `dto/openai_request_zero_value_test.go` 读到两个 PreserveExplicitZeroValues 测试用例结束。
3. 从 `common/json.go` 读完 `Marshal/Unmarshal/DecodeJson`。
阅读顺序：`dto/openai_request.go -> dto/claude.go -> dto/gemini.go -> dto/openai_request_zero_value_test.go -> common/json.go`。
读后检查题：字段缺失、字段为 `0`、字段为 `false` 三者语义有什么不同。

### 今天搞清楚什么
1. 为什么可选标量字段要用指针。
2. `omitempty` 对 `0/false` 的影响。
3. 请求结构与上游协议兼容风险。

### 今天实现什么
1. 设计 `ChatCompletionRequest` DTO（可选字段使用 `*int/*bool/*float64`）。
2. 统一JSON封装函数（封装 marshal/unmarshal）。
3. 参数校验（必填字段、范围、类型）。

### 今天怎么验证
1. 构造 `max_tokens=0`、`stream=false` 请求并重新序列化。
2. 确认字段没有被丢弃。
3. 为“显式零值保留”写单测。

### 今天通过标准
1. 你能解释“字段缺失”和“字段显式为0”的语义差异。
2. 你能独立修复 `omitempty` 导致的零值丢失bug。

---

## Day 5：Relay主流程框架

### 今天先读源码（阅读边界）
1. 从 `controller/relay.go` 的 `Relay()` 函数开头读到重试循环结束。
2. 从 `relay/helper/valid_request.go` 的 `GetAndValidateRequest()` 读到各分支解析函数。
3. 从 `relay/common/relay_info.go` 的 `GenRelayInfo()` 读到 `InitChannelMeta()`。
阅读顺序：`controller/relay.go -> relay/helper/valid_request.go -> relay/common/relay_info.go -> relay/compatible_handler.go(TextHelper)`。
读后检查题：`Relay()` 里“解析请求、计费预扣、选路、上游调用、错误处理”先后顺序为什么不能乱。

### 今天搞清楚什么
1. 控制器主流程的顺序依赖。
2. 错误处理和返回结构统一的重要性。
3. 业务主链路与工具函数的边界。

### 今天实现什么
1. `Relay()` 框架：解析 -> 选路 -> 调上游 -> 回包。
2. 统一错误响应结构。
3. request context（保存 request_id、token_id、model、channel_id）。

### 今天怎么验证
1. 用 mock 上游接口跑通一次完整请求。
2. 上游失败时能返回结构化错误。
3. 日志能串起来同一个 request_id。

### 今天通过标准
1. 你能独立写出最小可用 `Relay()`。
2. 你能解释每一步的输入输出。

---

## Day 6：OpenAI非流式中转

### 今天先读源码（阅读边界）
1. 从 `relay/channel/openai/adaptor.go` 的 `Init()` 读到 `DoRequest()`。
2. 从 `relay/channel/openai/adaptor.go` 的 `DoResponse()` 读到 `relay/channel/openai/relay-openai.go` 的 `OpenaiHandler()`。
3. 从 `relay/compatible_handler.go` 的 `TextHelper()` 读到 `channel.DoApiRequest` 调用点。
阅读顺序：`relay/compatible_handler.go -> relay/channel/openai/adaptor.go -> relay/channel/openai/relay-openai.go`。
读后检查题：OpenAI路径下，URL、Header、Body 分别在哪一步组装。

### 今天搞清楚什么
1. OpenAI Chat Completions 请求格式。
2. 下游请求与上游请求字段透传规则。
3. header 组装策略（Authorization、Content-Type）。

### 今天实现什么
1. OpenAI adaptor（非流式）。
2. `/v1/chat/completions` 到上游请求转发。
3. 上游响应透传并做最小字段补齐。

### 今天怎么验证
1. 用真实上游或本地mock返回完整回答。
2. 错误码和错误信息能透出。
3. 至少覆盖一个异常测试。

### 今天通过标准
1. 你能在不看资料下描述一次完整OpenAI转发。
2. 你能解释哪些字段必须透传，哪些字段可忽略。

---

## Day 7：Claude协议兼容

### 今天先读源码（阅读边界）
1. 从 `relay/claude_handler.go` 的 `ClaudeHelper()` 读到适配器调用结束。
2. 从 `relay/channel/claude/adaptor.go` 的 `ConvertOpenAIRequest()` 读到 `DoResponse()`。
3. 从 `service/convert.go` 的 `ClaudeToOpenAIRequest()` 读到 `ResponseOpenAI2Claude()`。
阅读顺序：`dto/claude.go -> relay/claude_handler.go -> relay/channel/claude/adaptor.go -> service/convert.go`。
读后检查题：OpenAI消息转Claude时，`system/messages/tools` 三块分别如何映射。

### 今天搞清楚什么
1. OpenAI消息格式与Claude消息格式差异。
2. role/content/tool调用在两协议中的映射关系。

### 今天实现什么
1. Claude adaptor。
2. OpenAI风格输入转换到Claude请求。
3. `/v1/messages` 路由打通。

### 今天怎么验证
1. 基础文本请求转换正确。
2. 系统提示词和用户消息顺序正确。
3. 转换失败时有明确错误信息。

### 今天通过标准
1. 你能列出最关键的3个协议差异点。
2. 你能手写一个最小转换函数。

---

## Day 8：Gemini协议兼容

### 今天先读源码（阅读边界）
1. 从 `relay/gemini_handler.go` 的 `GeminiHelper()` 读到 `GeminiEmbeddingHandler()`。
2. 从 `relay/channel/gemini/adaptor.go` 的 `ConvertOpenAIRequest()` 读到 `DoResponse()`。
3. 从 `service/convert.go` 的 `GeminiToOpenAIRequest()` 读到 `ResponseOpenAI2Gemini()`。
阅读顺序：`dto/gemini.go -> relay/gemini_handler.go -> relay/channel/gemini/adaptor.go -> service/convert.go`。
读后检查题：Gemini 的 path/action 与 OpenAI endpoint 的映射是在哪一层完成的。

### 今天搞清楚什么
1. Gemini路径风格和OpenAI路径风格差异。
2. content/parts 结构与消息结构映射。

### 今天实现什么
1. Gemini adaptor。
2. OpenAI到Gemini请求转换（文本优先）。
3. `/v1beta/models/*` 路由支持。

### 今天怎么验证
1. 模型路径解析正确。
2. 文本请求可正常返回。
3. 不支持字段有降级策略或错误提示。

### 今天通过标准
1. 你能解释Gemini请求路径中 `:action` 的用途。
2. 你能说明转换层应放在哪一层最合理。

---

## Day 9：SSE流式回传

### 今天先读源码（阅读边界）
1. 从 `relay/channel/openai/relay-openai.go` 的 `OaiStreamHandler()` 读到 `sendStreamData()`。
2. 从 `relay/helper/stream_scanner.go` 的 `StreamScannerHandler()` 读完。
3. 从 `relay/common/stream_status.go` 读完，并对照 `relay/common/stream_status_test.go`。
阅读顺序：`relay/channel/openai/relay-openai.go -> relay/helper/stream_scanner.go -> relay/common/stream_status.go -> relay/common/stream_status_test.go`。
读后检查题：流式结束标记和错误记录如何保证“只记一次结束原因”。

### 今天搞清楚什么
1. SSE基本格式和分块输出方式。
2. 流式结束标识与错误事件处理。

### 今天实现什么
1. `stream=true` 分支处理。
2. 上游流式响应读取并转发到客户端。
3. 标准结束事件和超时中断处理。

### 今天怎么验证
1. 客户端能持续收到 chunk。
2. 最后有结束标志。
3. 中断时有可读错误，不挂死连接。

### 今天通过标准
1. 你能解释阻塞读取和流式flush的关系。
2. 你能定位常见“只返回首块/末块”的问题。

---

## Day 10：重试与容错

### 今天先读源码（阅读边界）
1. 从 `controller/relay.go` 的重试 `for` 循环读到 `shouldRetry()`、`processChannelError()`。
2. 从 `service/channel_select.go` 的 `RetryParam` 读到 `CacheGetRandomSatisfiedChannel()`。
3. 从 `model/channel.go` 的 `GetNextEnabledKey()` 读到 `UpdateChannelStatus()`。
阅读顺序：`controller/relay.go -> service/channel_select.go -> model/channel.go -> service/error.go(RelayErrorHandler)`。
读后检查题：哪些错误会立即停止重试，哪些错误会继续重试。

### 今天搞清楚什么
1. 哪些错误值得重试，哪些不该重试。
2. 重试上限和超时预算的关系。
3. 渠道失败与key失败的区别。

### 今天实现什么
1. 超时控制。
2. 失败重试策略（最多N次）。
3. 备用渠道切换和多key轮换。

### 今天怎么验证
1. 人工注入上游500，能触发重试。
2. 主渠道不可用时能切备用渠道。
3. 超过重试次数后返回最终错误。

### 今天通过标准
1. 你能说清“重试会导致什么副作用”。
2. 你能解释你的重试停止条件。

---

## Day 11：计费用量闭环

### 今天先读源码（阅读边界）
1. 从 `service/billing.go` 的 `PreConsumeBilling()` 读到 `SettleBilling()`。
2. 从 `service/billing_session.go` 的 `NewBillingSession()` 读到 `Settle()`、`Refund()`。
3. 从 `service/quota.go` 的 `PreConsumeTokenQuota()` 读到 `PostConsumeQuota()`。
阅读顺序：`relay/common/billing.go -> service/billing.go -> service/billing_session.go -> service/quota.go -> service/task_billing.go`。
读后检查题：为什么结算与退款要做成幂等。

### 今天搞清楚什么
1. 预扣费、结算、失败退款三段式。
2. 为什么要记录“预扣量”和“实际量”。
3. 如何避免重复扣费。

### 今天实现什么
1. `UsageRecord`、`BillingSession`。
2. 成功结算和失败回滚逻辑。
3. 请求级幂等标识（避免重复结算）。

### 今天怎么验证
1. 成功请求：预扣后正确结算。
2. 失败请求：预扣可回滚。
3. 重复回调不会重复扣费。

### 今天通过标准
1. 你能画出账务状态机。
2. 你能解释为什么计费逻辑不能散落在各handler。

---

## Day 12：稳定性保护

### 今天先读源码（阅读边界）
1. 从 `middleware/rate-limit.go` 的 `GlobalAPIRateLimit()` 读到 `SearchRateLimit()`。
2. 从 `middleware/model-rate-limit.go` 的 `ModelRequestRateLimit()` 读到底层redis/memory实现。
3. 从 `common/gin.go` 的 `UnmarshalBodyReusable()` 读到请求体大小限制相关逻辑，再看 `controller/relay.go` 的 413 处理分支。
阅读顺序：`middleware/rate-limit.go -> middleware/model-rate-limit.go -> common/gin.go -> middleware/gzip.go -> service/sensitive.go -> setting/sensitive.go`。
读后检查题：请求体超限为什么要统一返回413，而不是400。

### 今天搞清楚什么
1. 限流作用点（全局、token、模型）。
2. 请求体大小限制为何必要。
3. 敏感词检测应在链路什么位置。

### 今天实现什么
1. 限流中间件。
2. 请求体大小限制。
3. 基础敏感词过滤。

### 今天怎么验证
1. 压测触发限流后返回429。
2. 大请求被拒绝且不会压垮服务。
3. 命中敏感词能被拦截。

### 今天通过标准
1. 你能解释误杀和漏检的平衡。
2. 你能说明这三项保护的先后顺序。

---

## Day 13：管理接口最小可用

### 今天先读源码（阅读边界）
1. 从 `router/api-router.go` 读 `/api/user`、`/api/token`、`/api/channel`、`/api/log` 路由组。
2. 从 `controller/user.go`、`controller/token.go`、`controller/channel.go`、`controller/log.go` 各挑一个“增删改查”入口读到 model 调用处。
3. 从 `model/user.go`、`model/token.go`、`model/channel.go`、`model/log.go` 看对应数据层函数。
阅读顺序：`router/api-router.go -> controller/*.go -> model/*.go`（同名资源成对阅读）。
读后检查题：哪些接口必须 Admin/Root 才能调用，普通用户为什么不该有权限。

### 今天搞清楚什么
1. 平台运营最小闭环需要哪些管理能力。
2. 管理接口鉴权与普通接口鉴权差异。

### 今天实现什么
1. `/api/user/*` 最小接口。
2. `/api/token/*` 最小接口。
3. `/api/channel/*`、`/api/log/*` 最小接口。

### 今天怎么验证
1. 能创建用户并生成token。
2. 能增删改查渠道。
3. 能查询关键调用日志。

### 今天通过标准
1. 你能说清“用户视角接口”和“管理员接口”边界。
2. 你能解释为什么要做最小可用而不是一次做全。

---

## Day 14：可观测性与错误治理

### 今天先读源码（阅读边界）
1. 从 `middleware/request-id.go` 的 `RequestId()` 读到 `logger/logHelper(...)` 的上下文字段拼装。
2. 从 `logger/logger.go` 读 `SetupLogger()`、`LogInfo/LogWarn/LogError()`。
3. 从 `types/error.go` 的 `NewError()` 读到 `ToOpenAIError()`、`IsSkipRetryError()`。
阅读顺序：`middleware/request-id.go -> middleware/logger.go -> logger/logger.go -> types/error.go -> service/error.go`。
读后检查题：同一个失败请求，如何通过 request_id + error_code 快速定位根因。

### 今天搞清楚什么
1. 结构化日志字段设计。
2. 错误码规范为何比错误字符串更重要。
3. 最小指标集合是什么。

### 今天实现什么
1. 统一日志格式（至少包含 request_id、token_id、channel_id、latency、status）。
2. 统一错误码映射。
3. 关键指标统计（请求数、错误数、重试数、平均延迟）。

### 今天怎么验证
1. 通过 request_id 能复盘单次请求全链路。
2. 错误统计能区分4xx与5xx。
3. 重试行为可见。

### 今天通过标准
1. 你能在5分钟内定位一次失败请求的根因。
2. 你能解释“没有观测就没有稳定性”。

---

## Day 15：交付与复现

### 今天先读源码（阅读边界）
1. 从 `common/init.go` 的 `InitEnv()` 读到所有关键环境变量初始化。
2. 从 `main.go` 的 `InitResources()` 回看启动依赖顺序。
3. 从 `Dockerfile`、`docker-compose.yml`、`.env.example`、`makefile` 读部署入口。
阅读顺序：`common/init.go -> main.go -> Dockerfile -> docker-compose.yml -> .env.example -> makefile`。
读后检查题：新机器部署失败时，优先排查哪些配置项和依赖项。

### 今天搞清楚什么
1. 可复现交付要素：环境、依赖、配置、验证脚本。
2. 从“能跑”到“可交付”的差异。

### 今天实现什么
1. Dockerfile 和 compose。
2. `.env.example` 完整化。
3. 端到端验收脚本（健康检查、鉴权、chat、流式、重试）。

### 今天怎么验证
1. 全新环境一键启动成功。
2. 验收脚本全部通过。
3. 交付文档可让别人独立跑起来。

### 今天通过标准
1. 你能在新机器1小时内完整部署。
2. 你能现场演示关键链路并解释设计取舍。

---

## 3. 每日复盘模板（强制）

创建 `learning-log/day-XX.md`，每天按这个模板填：

```md
# Day XX 复盘

## 今天我搞清楚了
1.
2.
3.

## 今天我实现了
1.
2.
3.

## 今天验证结果
- 命令:
- 结果:

## 今天踩坑
1.

## 明天开始前我还不清楚的问题
1.
```

---

## 4. 最终验收清单（第15天必须全绿）

1. 模型列表接口可用。
2. OpenAI非流式可用。
3. OpenAI流式可用。
4. Claude/Gemini兼容至少文本链路可用。
5. 失败重试与备用渠道切换可用。
6. 计费预扣、结算、失败回滚可用。
7. 管理接口可最小运营。
8. 请求日志可按 request_id 全链追踪。
9. Docker 一键部署成功。

如果清单未全绿，不算完成，继续补齐直到全绿。
