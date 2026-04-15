package common

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
)

// 程序启动参数
var (
	Port         = flag.Int("port", 3000, "the listening port")                  // 监听端口号
	PrintVersion = flag.Bool("version", false, "print version and exit")         // 版本号
	PrintHelp    = flag.Bool("help", false, "print help and exit")               // 帮助文档
	LogDir       = flag.String("log-dir", "./logs", "specify the log directory") // 日志目录
)

// 打印帮助文档
func printHelp() {
	fmt.Println("NewAPI(Based OneAPI) " + Version + " - The next-generation LLM gateway and AI asset management system supports multiple languages.")
	fmt.Println("Original Project: OneAPI by JustSong - https://github.com/songquanpeng/one-api")
	fmt.Println("Maintainer: QuantumNous - https://github.com/QuantumNous/new-api")
	fmt.Println("Usage: newapi [--port <port>] [--log-dir <log directory>] [--version] [--help]")
}

// 初始化环境信息
func InitEnv() {
	flag.Parse() // 解析命令行参数

	// 先从环境变量里读取 VERSION
	// 如果读到了非空值，就用它覆盖程序里的 Version
	// 如果没设置，就继续用原来的默认 Version
	envVersion := os.Getenv("VERSION")
	if envVersion != "" {
		Version = envVersion
	}

	// 如果是打印版本号, 就输出然后退出
	if *PrintVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	// 打印帮助文档, 退出
	if *PrintHelp {
		printHelp()
		os.Exit(0)
	}

	// 从环境变量中取`SESSION_SECRET`
	// 如果存在就判断是不是默认值`random_string`
	// 如果是默认值就打印警告,并退出程序
	// 否则把`SESSION_SECRET`复制给常量`SessionSecret`
	if os.Getenv("SESSION_SECRET") != "" {
		ss := os.Getenv("SESSION_SECRET")
		if ss == "random_string" {
			log.Println("WARNING: SESSION_SECRET is set to the default value 'random_string', please change it to a random string.")
			log.Println("警告：SESSION_SECRET被设置为默认值'random_string'，请修改为随机字符串。")
			log.Fatal("Please set SESSION_SECRET to a random string.")
		} else {
			SessionSecret = ss
		}
	}

	// 从环境变量中获取加密密钥
	// 如果没有就使用会话密钥
	if os.Getenv("CRYPTO_SECRET") != "" {
		CryptoSecret = os.Getenv("CRYPTO_SECRET")
	} else {
		CryptoSecret = SessionSecret
	}

	// 这里的SQLITE_PATH不是文件地址, 而是gorm连接数据库的dsn
	if os.Getenv("SQLITE_PATH") != "" {
		SQLitePath = os.Getenv("SQLITE_PATH")
	}

	// 这里的*LogDir永远不可能==""
	// 创建日志目录
	if *LogDir != "" {
		var err error
		*LogDir, err = filepath.Abs(*LogDir)
		if err != nil {
			log.Fatal(err)
		}
		if err = os.MkdirAll(*LogDir, 0o777); err != nil {
			log.Fatal(err)
		}
	}

	// 从 constants.go 初始化使用环境变量的变量
	// 是否开启调试模式
	DebugEnabled = os.Getenv("DEBUG") == "true"
	// 是否启用内存缓存
	// 开启后，程序可能会把一些常用数据放在内存里，减少数据库或 Redis 查询
	MemoryCacheEnabled = os.Getenv("MEMORY_CACHE_ENABLED") == "true"
	// 作用：判断当前节点是不是主节点。
	// 只要 NODE_TYPE=slave，就是从节点，IsMasterNode=false
	// 其他任何值，甚至没设置，都会被当成主节点，IsMasterNode=true

	// 通常用于区分部署角色：
	// 主节点：执行管理任务、写操作、后台任务、定时任务
	// 从节点：只提供部分服务，避免重复执行关键任务
	IsMasterNode = os.Getenv("NODE_TYPE") != "slave"

	// TLSInsecureSkipVerify 用来控制“当前程序作为 HTTP 客户端向外请求 HTTPS 地址时，
	// 是否跳过对对方服务器证书的校验”。
	//
	// 这里影响的是“我请求别人”的出站请求，例如：
	// 1. 请求上游模型服务；
	// 2. 请求第三方 API；
	// 3. 请求 OAuth / OIDC 提供商；
	// 4. 请求任何使用 HTTPS 的外部服务。
	//
	// 它不影响“别人请求我”的入站请求，也就是说：
	// 1. 不会修改 Gin 服务器的 TLS 行为；
	// 2. 不会改变浏览器或其他客户端访问本服务时的证书校验逻辑；
	// 3. 不会让本服务自动变成 HTTPS 服务端配置。
	//
	// 默认值是 false，表示默认开启正常的 TLS 安全校验。
	// 只有显式设置环境变量 TLS_INSECURE_SKIP_VERIFY=true 时才会跳过校验。
	TLSInsecureSkipVerify = GetEnvOrDefaultBool("TLS_INSECURE_SKIP_VERIFY", false)
	if TLSInsecureSkipVerify {
		// http.DefaultTransport 是 Go 标准库里“默认 HTTP 客户端”使用的传输层配置。
		// 很多没有自定义 http.Client / http.Transport 的请求，最终都会走到这里。
		//
		// 例如常见的默认调用方式：
		// 1. http.Get(...)
		// 2. http.Post(...)
		// 3. http.DefaultClient.Do(req)
		// 4. 某些未显式指定 Transport 的 HTTP 客户端
		//
		// 这里通过类型断言把 http.DefaultTransport 转成 *http.Transport，
		// 因为只有 *http.Transport 才有 TLSClientConfig 这个字段可以修改。
		//
		// ok 表示类型断言是否成功：
		// 1. ok == true 说明默认传输器确实是 *http.Transport；
		// 2. ok == false 说明它不是这个类型，此时不能继续改 TLS 配置。
		//
		// tr != nil 是额外的空值保护，避免空指针问题。
		if tr, ok := http.DefaultTransport.(*http.Transport); ok && tr != nil {
			// TLSClientConfig 是底层 TLS 配置，控制 HTTPS 握手时的行为。
			// 其中 InsecureSkipVerify=true 的含义是：
			// “跳过服务端证书链和主机名校验”。
			//
			// 开启后，这个程序在访问 HTTPS 地址时，即使对方证书存在以下问题，
			// 也可能继续连通：
			// 1. 自签名证书；
			// 2. 证书不是受信任 CA 签发；
			// 3. 证书主机名和访问域名不匹配；
			// 4. 其他证书校验异常。
			//
			// 这通常适用于：
			// 1. 本地开发；
			// 2. 内网测试；
			// 3. 对接证书配置不规范的测试环境。
			//
			// 但正式环境应尽量避免开启，因为这会降低 TLS 的安全性，
			// 增加中间人攻击风险。
			if tr.TLSClientConfig != nil {
				// 如果默认 Transport 已经有 TLSClientConfig，
				// 说明之前已经存在一份 TLS 配置对象。
				//
				// 这里不替换整份配置，而是只修改其中的 InsecureSkipVerify 字段，
				// 这样可以尽量保留原本其他 TLS 参数。
				tr.TLSClientConfig.InsecureSkipVerify = true
			} else {
				// 如果默认 Transport 还没有 TLSClientConfig，
				// 就直接挂上一份预先定义好的“不校验证书”配置。
				//
				// InsecureTLSConfig 通常会是类似下面的配置：
				// &tls.Config{InsecureSkipVerify: true}
				//
				// 这样之后所有走这个默认 Transport 的 HTTPS 出站请求，
				// 就都会按“跳过证书校验”的模式执行。
				tr.TLSClientConfig = InsecureTLSConfig
			}
		}
	}

	// 从环境变量读取轮询间隔（单位：秒），转换成 time.Duration 供程序使用。
	requestInterval, _ = strconv.Atoi(os.Getenv("POLLING_INTERVAL"))
	RequestInterval = time.Duration(requestInterval) * time.Second

	// 同步相关配置
	// SYNC_FREQUENCY：数据同步频率
	SyncFrequency = GetEnvOrDefault("SYNC_FREQUENCY", 60)
	// BATCH_UPDATE_INTERVAL：批量更新的执行间隔
	BatchUpdateInterval = GetEnvOrDefault("BATCH_UPDATE_INTERVAL", 5)
	// RELAY_TIMEOUT：中转请求超时时间
	RelayTimeout = GetEnvOrDefault("RELAY_TIMEOUT", 0)
	// RELAY_MAX_IDLE_CONNS：HTTP 连接池允许的最大空闲连接数
	RelayMaxIdleConns = GetEnvOrDefault("RELAY_MAX_IDLE_CONNS", 500)
	// RELAY_MAX_IDLE_CONNS_PER_HOST：每个主机允许保留的最大空闲连接数
	RelayMaxIdleConnsPerHost = GetEnvOrDefault("RELAY_MAX_IDLE_CONNS_PER_HOST", 100)

	// 不同上游服务的默认安全策略配置
	GeminiSafetySetting = GetEnvOrDefaultString("GEMINI_SAFETY_SETTING", "BLOCK_NONE")
	CohereSafetySetting = GetEnvOrDefaultString("COHERE_SAFETY_SETTING", "NONE")

	// 全局 API 限流配置
	GlobalApiRateLimitEnable = GetEnvOrDefaultBool("GLOBAL_API_RATE_LIMIT_ENABLE", true)
	GlobalApiRateLimitNum = GetEnvOrDefault("GLOBAL_API_RATE_LIMIT", 180)
	GlobalApiRateLimitDuration = int64(GetEnvOrDefault("GLOBAL_API_RATE_LIMIT_DURATION", 180))

	// 全局 Web 页面访问限流配置
	GlobalWebRateLimitEnable = GetEnvOrDefaultBool("GLOBAL_WEB_RATE_LIMIT_ENABLE", true)
	GlobalWebRateLimitNum = GetEnvOrDefault("GLOBAL_WEB_RATE_LIMIT", 60)
	GlobalWebRateLimitDuration = int64(GetEnvOrDefault("GLOBAL_WEB_RATE_LIMIT_DURATION", 180))

	// 关键操作的限流配置
	CriticalRateLimitEnable = GetEnvOrDefaultBool("CRITICAL_RATE_LIMIT_ENABLE", true)
	CriticalRateLimitNum = GetEnvOrDefault("CRITICAL_RATE_LIMIT", 20)
	CriticalRateLimitDuration = int64(GetEnvOrDefault("CRITICAL_RATE_LIMIT_DURATION", 20*60))

	// 搜索相关接口的限流配置
	SearchRateLimitEnable = GetEnvOrDefaultBool("SEARCH_RATE_LIMIT_ENABLE", true)
	SearchRateLimitNum = GetEnvOrDefault("SEARCH_RATE_LIMIT", 10)
	SearchRateLimitDuration = int64(GetEnvOrDefault("SEARCH_RATE_LIMIT_DURATION", 60))

	// 初始化 constant 包里的环境变量配置
	initConstantEnv()
}

// 暂时先不研究, 后面用到的时候再来看
func initConstantEnv() {
	constant.StreamingTimeout = GetEnvOrDefault("STREAMING_TIMEOUT", 300)
	constant.DifyDebug = GetEnvOrDefaultBool("DIFY_DEBUG", true)
	constant.MaxFileDownloadMB = GetEnvOrDefault("MAX_FILE_DOWNLOAD_MB", 64)
	constant.StreamScannerMaxBufferMB = GetEnvOrDefault("STREAM_SCANNER_MAX_BUFFER_MB", 128)
	// MaxRequestBodyMB 请求体最大大小（解压后），用于防止超大请求/zip bomb导致内存暴涨
	constant.MaxRequestBodyMB = GetEnvOrDefault("MAX_REQUEST_BODY_MB", 128)
	// ForceStreamOption 覆盖请求参数，强制返回usage信息
	constant.ForceStreamOption = GetEnvOrDefaultBool("FORCE_STREAM_OPTION", true)
	constant.CountToken = GetEnvOrDefaultBool("CountToken", true)
	constant.GetMediaToken = GetEnvOrDefaultBool("GET_MEDIA_TOKEN", true)
	constant.GetMediaTokenNotStream = GetEnvOrDefaultBool("GET_MEDIA_TOKEN_NOT_STREAM", false)
	constant.UpdateTask = GetEnvOrDefaultBool("UPDATE_TASK", true)
	constant.AzureDefaultAPIVersion = GetEnvOrDefaultString("AZURE_DEFAULT_API_VERSION", "2025-04-01-preview")
	constant.NotifyLimitCount = GetEnvOrDefault("NOTIFY_LIMIT_COUNT", 2)
	constant.NotificationLimitDurationMinute = GetEnvOrDefault("NOTIFICATION_LIMIT_DURATION_MINUTE", 10)
	// GenerateDefaultToken 是否生成初始令牌，默认关闭。
	constant.GenerateDefaultToken = GetEnvOrDefaultBool("GENERATE_DEFAULT_TOKEN", false)
	// 是否启用错误日志
	constant.ErrorLogEnabled = GetEnvOrDefaultBool("ERROR_LOG_ENABLED", false)
	// 任务轮询时查询的最大数量
	constant.TaskQueryLimit = GetEnvOrDefault("TASK_QUERY_LIMIT", 1000)
	// 异步任务超时时间（分钟），超过此时间未完成的任务将被标记为失败并退款。0 表示禁用。
	constant.TaskTimeoutMinutes = GetEnvOrDefault("TASK_TIMEOUT_MINUTES", 1440)

	soraPatchStr := GetEnvOrDefaultString("TASK_PRICE_PATCH", "")
	if soraPatchStr != "" {
		var taskPricePatches []string
		soraPatches := strings.Split(soraPatchStr, ",")
		for _, patch := range soraPatches {
			trimmedPatch := strings.TrimSpace(patch)
			if trimmedPatch != "" {
				taskPricePatches = append(taskPricePatches, trimmedPatch)
			}
		}
		constant.TaskPricePatches = taskPricePatches
	}

	// Initialize trusted redirect domains for URL validation
	trustedDomainsStr := GetEnvOrDefaultString("TRUSTED_REDIRECT_DOMAINS", "")
	var trustedDomains []string
	domains := strings.Split(trustedDomainsStr, ",")
	for _, domain := range domains {
		trimmedDomain := strings.TrimSpace(domain)
		if trimmedDomain != "" {
			// Normalize domain to lowercase
			trustedDomains = append(trustedDomains, strings.ToLower(trimmedDomain))
		}
	}
	constant.TrustedRedirectDomains = trustedDomains
}
