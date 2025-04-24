package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gin-contrib/cors"  // 跨域资源共享中间件
	"github.com/gin-gonic/gin"     // Web框架
	"github.com/golang-jwt/jwt/v5" // JWT库
	"github.com/jmoiron/sqlx"      // 增强的数据库操作包

	// 注册数据库驱动，但不直接使用其中的函数
	_ "github.com/denisenkom/go-mssqldb" // SQL Server驱动
	_ "github.com/go-sql-driver/mysql"   // MySQL驱动
	_ "github.com/lib/pq"                // PostgreSQL驱动
)

type (
	// Map 是一个通用的键值对映射类型，用于存储任意类型的值
	Map map[string]any
	// Cfg 定义配置文件结构
	Cfg struct {
		Driver    string `json:"driver"`    // 数据库驱动类型：mssql/mysql/postgres
		Dsn       string `json:"dsn"`       // 数据库连接字符串
		Query     string `json:"query"`     // 用于获取SQL模板的查询语句
		Api       string `json:"api"`       // API路由路径
		Port      int    `json:"port"`      // 服务监听端口
		JwtSecret string `json:"jwtSecret"` // JWT密钥
		JwtExpire int64  `json:"jwtExpire"` // JWT过期时间（秒）
		JwtIssuer string `json:"jwtIssuer"` // JWT签发者

		WechatAppID    string `json:"wechatAppID"`    // 微信小程序AppID
		WechatSecret   string `json:"wechatSecret"`   // 微信小程序Secret
		WechatTokenUrl string `json:"wechatTokenUrl"` // 微信获取token的URL

		AuthQuery      string            `json:"authQuery"`      // 鉴权查询语句
		Routes         map[string]string `json:"routes"`         // 路由映射配置
		QueryTemplates map[string]string `json:"queryTemplates"` // 查询模板名称映射
	}

	// JwtClaims 定义JWT的载荷结构
	JwtClaims struct {
		UserID   int    `json:"user_id"`
		Username string `json:"username"`
		jwt.RegisteredClaims
	}
)

var (
	db      *sqlx.DB   // 数据库连接实例
	cfg     = new(Cfg) // 配置实例
	dbMutex sync.Mutex // 保护数据库连接操作的互斥锁
)

// main 程序入口函数
func main() {
	// 读取与可执行文件同名的JSON配置文件
	fp, fn := filepath.Split(os.Args[0])
	b, err := ioutil.ReadFile(fp + strings.Replace(fn, ".exe", ".json", 1))
	CatchErr("READ-CONF:", err)
	CatchErr("PARSE-CONF:", json.Unmarshal(b, &cfg))

	// 初始化数据库连接池
	initDB()

	// 设置Gin为发布模式，减少日志输出
	gin.SetMode(gin.ReleaseMode)
	// 创建默认的Gin路由引擎，包含Logger和Recovery中间件
	r := gin.Default()
	// 添加CORS中间件，允许跨域请求
	r.Use(cors.Default())
	// 添加JWT认证中间件
	r.Use(JWTAuth())
	// 注册通用API处理函数，支持所有HTTP方法
	r.Any(cfg.Api, Api)

	// 使用配置的路由路径注册登录接口
	loginPath := "/login" // 默认路径
	if cfg.Routes != nil && cfg.Routes["login"] != "" {
		loginPath = cfg.Routes["login"]
	}
	r.POST(loginPath, NxLogin)

	// 使用配置的路由路径注册微信登录接口
	wxLoginPath := "/wx/login" // 默认路径
	if cfg.Routes != nil && cfg.Routes["wxLogin"] != "" {
		wxLoginPath = cfg.Routes["wxLogin"]
	}
	r.POST(wxLoginPath, WxLogin)

	// 打印启动信息
	log.Printf("【慧工厂】·【API启动:%v】·【by 一零院长】·【2023-present】·【v250424】", cfg.Port)
	// 启动HTTP服务
	r.Run(fmt.Sprint(":", cfg.Port))
}

// initDB 初始化并配置数据库连接池
func initDB() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	var err error
	// 连接数据库
	db, err = sqlx.Connect(cfg.Driver, cfg.Dsn)
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(25)                 // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期

	// 启动一个goroutine定期检查数据库连接
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			if err := db.Ping(); err != nil {
				log.Println("数据库连接丢失，尝试重连...")
				reconnectDB()
			}
		}
	}()
}

// reconnectDB 尝试重新连接数据库
func reconnectDB() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	var err error
	// 最多尝试5次重连
	for i := 0; i < 5; i++ {
		db, err = sqlx.Connect(cfg.Driver, cfg.Dsn)
		if err == nil {
			log.Println("数据库重连成功")
			return
		}
		time.Sleep(5 * time.Second)
	}
	log.Fatalf("无法重连数据库: %v", err)
}

// Api 是通用的API处理函数，处理所有API请求
func Api(c *gin.Context) {
	// 解析请求参数
	param := ParseForm(c)
	log.Println("REQ:", param)

	// 获取路由参数和HTTP方法
	action := c.Param("a")     // 从路由路径中提取动作参数
	method := c.Request.Method // 获取HTTP方法(GET/POST等)
	sqlstr := cfg.Query        // 获取SQL模板查询语句

	// 从数据库获取SQL模板和鉴权信息
	var result struct {
		模板 string `db:"模板"`
		鉴权 *int   `db:"鉴权"` // 使用指针类型，以便能够处理NULL值
	}

	err := db.Get(&result, sqlstr, action, method)
	if err != nil {
		c.JSON(http.StatusNotFound, Map{"status": 404, "msg": "API未找到"})
		return
	}

	tmpStr := result.模板
	buf := new(bytes.Buffer)
	tmpsql := tmpStr

	// 解析SQL模板
	tmp, e := template.New(action + method).Parse(tmpStr)
	if e != nil {
		// 模板解析失败，返回错误
		c.JSON(http.StatusOK, Map{"data": e, "status": 1})
		return
	}
	// 将请求参数应用到模板
	if e = tmp.Execute(buf, param); e == nil {
		tmpsql = buf.String()
	}
	log.Println("TMP:", tmpsql, e)

	// 执行SQL查询
	data := make([]Map, 0)
	rows, err := db.Queryx(tmpsql)
	CatchErr("QUERY-ERR:", err)
	// 处理查询结果
	for rows.Next() {
		mp := make(Map, 0)
		err = rows.MapScan(mp)
		if err == nil {
			// 转换值的格式
			for k, val := range mp {
				mp[k] = Conv(val)
			}
			data = append(data, mp)
		}
	}
	log.Println("RET:", data)
	// 返回JSON格式的结果
	c.JSON(http.StatusOK, Map{"data": data, "status": 0})
}

// Conv 转换数据库查询结果中的值为更适合JSON格式的类型
func Conv(pval interface{}) interface{} {
	switch v := (pval).(type) {
	case nil:
		return "" // 将null值转为空字符串
	case []byte:
		return string(v) // 将字节数组转为字符串
	case time.Time:
		return v.Format("2006-01-02 15:04:05") // 格式化时间类型
	default:
		return v // 其他类型保持不变
	}
}

// ParseForm 解析HTTP请求中的参数
func ParseForm(c *gin.Context) Map {
	c.Request.ParseForm() // 解析URL查询参数和表单数据
	param := make(Map)

	// 处理非GET和非DELETE请求的请求体
	if c.Request.Method != "GET" && c.Request.Method != "DELETE" {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err == nil {
			c.Set("body", body) // 将原始请求体存储在上下文中
		}
		// 尝试将请求体解析为JSON
		CatchErr("BIND-BODY", json.Unmarshal([]byte(body), &param))
	}

	// 处理URL查询参数和表单数据
	for k, v := range c.Request.Form {
		param[k] = v[0] // 只取每个参数的第一个值
		// 尝试URL解码
		if pp, err := url.QueryUnescape(v[0]); err == nil {
			param[k] = pp
		}
	}

	return param
}

// CatchErr 简单的错误处理函数，记录错误但不中断执行
func CatchErr(desc string, err error) {
	if err != nil {
		log.Println(desc, err)
	}
}

// 生成JWT令牌
func GenerateToken(userID int, username string) (string, error) {
	// 设置JWT声明
	claims := JwtClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(cfg.JwtExpire))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.JwtIssuer,
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	return token.SignedString([]byte(cfg.JwtSecret))
}

// 解析并验证JWT令牌
func ParseToken(tokenString string) (*JwtClaims, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// 验证令牌有效性
	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的令牌")
}

// JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路由获取Action参数
		action := c.Param("a")

		// 获取配置的登录路径
		loginPath := "/login"
		if cfg.Routes != nil && cfg.Routes["login"] != "" {
			loginPath = cfg.Routes["login"]
		}

		// 获取配置的微信登录路径
		wxLoginPath := "/wx/login"
		if cfg.Routes != nil && cfg.Routes["wxLogin"] != "" {
			wxLoginPath = cfg.Routes["wxLogin"]
		}

		// 如果是登录接口，跳过认证
		if c.FullPath() == loginPath || c.FullPath() == wxLoginPath {
			c.Next()
			return
		}

		// 查询此API的模板和鉴权信息
		var result struct {
			模板 string `db:"模板"`
			鉴权 *int   `db:"鉴权"` // 使用指针类型，以便能够处理NULL值
		}

		// 使用配置中的查询语句
		err := db.Get(&result, cfg.Query, action, c.Request.Method)

		if err != nil {
			// 查询出错，默认为匿名访问，继续处理请求
			log.Println("API查询出错:", err)
			c.Next()
			return
		}

		// 处理鉴权为NULL的情况
		if result.鉴权 == nil {
			log.Println("API鉴权值为NULL:", action)
			c.Next()
			return
		}

		// 根据鉴权标志值处理
		switch *result.鉴权 {
		case 0: // 匿名访问
			c.Next()
			return
		case 1: // 需要鉴权
			// 获取请求头中的Authorization
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": "未提供授权令牌"})
				c.Abort()
				return
			}

			// 处理Bearer令牌格式
			parts := strings.SplitN(authHeader, " ", 2)
			if !(len(parts) == 2 && parts[0] == "Bearer") {
				c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": "令牌格式错误"})
				c.Abort()
				return
			}

			// 解析令牌
			claims, err := ParseToken(parts[1])
			if err != nil {
				c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": "无效令牌: " + err.Error()})
				c.Abort()
				return
			}

			// 将用户信息存储在上下文中，供后续处理使用
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Next()
		default: // 未知的鉴权类型
			c.JSON(http.StatusForbidden, Map{"status": 403, "msg": "鉴权类型未知"})
			c.Abort()
			return
		}
	}
}

// MD5 计算字符串的MD5哈希值
func MD5(text string) string {
	hash := md5.New()
	hash.Write([]byte(text))
	return hex.EncodeToString(hash.Sum(nil))
}
