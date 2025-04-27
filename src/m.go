package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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
		Driver string `json:"driver"` // 数据库驱动类型：mssql/mysql/postgres
		Dsn    string `json:"dsn"`    // 数据库连接字符串
		Query  string `json:"query"`  // 用于获取SQL模板的查询语句
		Api    string `json:"api"`    // API路由路径
		Port   int    `json:"port"`   // 服务监听端口

		JWTSecret string `json:"jwtSecret"` // JWT签名密钥
		JWTExpire int    `json:"jwtExpire"` // JWT过期时间（秒）
		JWTIssuer string `json:"jwtIssuer"` // JWT签发者

		WechatAppID          string `json:"wechatAppID"`          // 微信小程序AppID
		WechatSecret         string `json:"wechatSecret"`         // 微信小程序Secret
		WechatTokenUrl       string `json:"wechatTokenUrl"`       // 微信接口URL
		WechatAccessTokenUrl string `json:"wechatAccessTokenUrl"` // 微信获取access_token接口URL
		WechatTicketUrl      string `json:"wechatTicketUrl"`      // 微信获取jsapi_ticket接口URL
	}
)

var (
	db      *sqlx.DB   // 数据库连接实例
	cfg     = new(Cfg) // 配置实例
	dbMutex sync.Mutex // 保护数据库连接操作的互斥锁
)

// Claims 定义JWT的声明
type Claims struct {
	UserID   int    `json:"userID"`
	UserName string `json:"userName"`
	jwt.RegisteredClaims
}

// 生成JWT令牌
func GenerateToken(userID int, userName string) (string, error) {
	// 设置过期时间
	expireTime := time.Now().Add(time.Duration(cfg.JWTExpire) * time.Second)

	claims := Claims{
		UserID:   userID,
		UserName: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.JWTIssuer,
		},
	}

	// 使用HS256算法创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名token
	return token.SignedString([]byte(cfg.JWTSecret))
}

// 验证JWT令牌
func ParseToken(tokenString string) (*Claims, error) {
	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法是否为HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// 验证并返回Claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// JWT授权中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			// 尝试从查询参数获取token
			tokenString = c.Query("token")
		}

		// 检查token是否存在
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, Map{"status": 1, "message": "未提供授权令牌"})
			c.Abort() // 只需要Abort，不需要return
			return
		}

		// 移除Bearer前缀（如果存在）
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}

		// 解析和验证token
		claims, err := ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, Map{"status": 1, "message": "无效的授权令牌", "error": err.Error()})
			c.Abort() // 只需要Abort，不需要return
			return
		}

		// 将用户信息存储在上下文中
		c.Set("userID", claims.UserID)
		c.Set("userName", claims.UserName)

		c.Next()
	}
}

// WechatResponse 微信登录接口返回数据结构
type WechatResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// 微信鉴权，通过code获取openid
func GetWechatOpenID(code string) (*WechatResponse, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		cfg.WechatTokenUrl, cfg.WechatAppID, cfg.WechatSecret, code)

	// 发起HTTP请求
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var wxResp WechatResponse
	if err := json.Unmarshal(body, &wxResp); err != nil {
		return nil, err
	}

	// 检查是否返回错误
	if wxResp.ErrCode != 0 {
		return nil, fmt.Errorf("微信接口返回错误: %d %s", wxResp.ErrCode, wxResp.ErrMsg)
	}

	return &wxResp, nil
}

// 用于存储jsapi_ticket和过期时间
var (
	jsapiTicket     string
	jsapiTicketTime time.Time
	jsapiTicketLock sync.Mutex
)

// 获取jsapi_ticket
func getJsapiTicket() (string, error) {
	jsapiTicketLock.Lock()
	defer jsapiTicketLock.Unlock()

	// 如果jsapi_ticket还有效，直接返回
	if jsapiTicket != "" && time.Since(jsapiTicketTime) < (time.Duration(7000)*time.Second) {
		return jsapiTicket, nil
	}

	// 第一步：请求access_token
	accessTokenUrl := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s", cfg.WechatAccessTokenUrl, cfg.WechatAppID, cfg.WechatSecret)
	resp, err := http.Get(accessTokenUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("获取access_token失败: %v", tokenResp.ErrMsg)
	}

	// 第二步：请求jsapi_ticket
	ticketUrl := fmt.Sprintf("%s?access_token=%s&type=jsapi", cfg.WechatTicketUrl, tokenResp.AccessToken)
	resp2, err := http.Get(ticketUrl)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()

	body2, _ := ioutil.ReadAll(resp2.Body)
	var ticketResp struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
	}
	if err := json.Unmarshal(body2, &ticketResp); err != nil {
		return "", err
	}
	if ticketResp.ErrCode != 0 {
		return "", fmt.Errorf("获取ticket失败: %v", ticketResp.ErrMsg)
	}

	jsapiTicket = ticketResp.Ticket
	jsapiTicketTime = time.Now()

	return jsapiTicket, nil
}

// 生成随机字符串
func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

// 生成微信JS-SDK签名
func generateWechatSignature(ticket, nonceStr string, timestamp int64, url string) string {
	str := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticket, nonceStr, timestamp, url)

	h := sha1.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// main 程序入口函数
func main() {
	// 读取与可执行文件同名的JSON配置文件
	fp, fn := filepath.Split(os.Args[0])
	b, err := ioutil.ReadFile(fp + strings.Replace(fn, ".exe", ".json", 1))
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}

	// 解析配置文件
	if err := json.Unmarshal(b, &cfg); err != nil {
		log.Fatalf("配置文件格式错误: %v", err)
	}

	// 初始化数据库连接池
	initDB()

	// 设置Gin为发布模式，减少日志输出
	gin.SetMode(gin.ReleaseMode)
	// 创建默认的Gin路由引擎，包含Logger和Recovery中间件
	r := gin.Default()

	// 配置CORS中间件
	r.Use(configureCORS())

	// API路由组，根据鉴权需求配置
	apiGroup := r.Group("/")

	// 注册通用API处理函数，支持所有HTTP方法
	apiGroup.Any(cfg.Api, Api)

	// 打印启动信息
	log.Printf("【慧工厂】·【API启动:%v】·【by 一零院长】·【2023-present】·【v250425】", cfg.Port)
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

// 验证密码 - ERP特殊密码验证
// 根据md5(md5(LoginName+Password)+salt)进行验证
func ValidatePassword(loginName, password, dbPassword, salt string) bool {
	// 第一步：计算md5(LoginName+Password)
	innerMd5 := md5.Sum([]byte(loginName + password))
	innerMd5Str := hex.EncodeToString(innerMd5[:])

	// 第二步：计算md5(md5(LoginName+Password)+salt)
	outerMd5 := md5.Sum([]byte(innerMd5Str + salt))
	calculatedPassword := hex.EncodeToString(outerMd5[:])

	// 比较计算出的密码与数据库中的密码
	return calculatedPassword == dbPassword
}

// Api 是通用的API处理函数，处理所有API请求
func Api(c *gin.Context) {
	// 允许跨域预检请求直接通过
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusNoContent)
		return
	}

	// 解析请求参数
	param := ParseForm(c)

	// 获取路由参数和HTTP方法
	action := c.Param("a")     // 从路由路径中提取动作参数
	method := c.Request.Method // 获取HTTP方法(GET/POST等)
	// 特殊处理微信签名请求
	if action == "wechat_signature" && method == "GET" {
		url := c.Query("url")
		if url == "" {
			c.JSON(http.StatusBadRequest, Map{"status": 1, "message": "缺少url参数"})
			return
		}

		ticket, err := getJsapiTicket()
		if err != nil {
			c.JSON(http.StatusInternalServerError, Map{"status": 1, "message": "获取jsapi_ticket失败", "error": err.Error()})
			return
		}

		nonceStr := randomString(16)
		timestamp := time.Now().Unix()

		signature := generateWechatSignature(ticket, nonceStr, timestamp, url)

		c.JSON(http.StatusOK, Map{
			"status":    0,
			"appId":     cfg.WechatAppID,
			"timestamp": timestamp,
			"nonceStr":  nonceStr,
			"signature": signature,
		})
		return
	}

	// 从数据库获取SQL模板和鉴权信息
	var tmpStr string
	var auth *int
	row := db.QueryRow(cfg.Query, action, method)
	err := row.Scan(&tmpStr, &auth)
	CatchErr("GET-API:", err)
	if err != nil {
		c.JSON(http.StatusNotFound, Map{"status": 1, "message": "API不存在"})
		return
	}

	// 检查是否需要鉴权
	if auth != nil && *auth == 1 {
		// 从请求头获取token
		tokenString := c.GetHeader("Authorization")

		if tokenString == "" {
			// 尝试从查询参数获取token
			tokenString = c.Query("token")
		}

		// 检查token是否存在
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, Map{"status": 1, "message": "需要授权令牌"})
			return
		}

		// 移除Bearer前缀（如果存在）
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}

		// 解析和验证token
		claims, err := ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, Map{"status": 1, "message": "无效的授权令牌", "error": err.Error()})
			return
		}

		// 将用户信息添加到参数中，以便SQL模板使用
		param["userID"] = claims.UserID
		param["userName"] = claims.UserName
	}

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

	// 执行SQL查询
	data := make([]Map, 0)
	rows, err := db.Queryx(tmpsql)
	CatchErr("QUERY-ERR:", err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 1, "message": "查询执行失败", "error": err.Error()})
		return
	}

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

	// 处理微信登录请求
	if action == "wxlogin" && method == "POST" {
		code, hasCode := param["code"]
		if hasCode {
			// 获取微信OpenID
			wxResp, err := GetWechatOpenID(code.(string))
			if err != nil {
				c.JSON(http.StatusOK, Map{
					"status":  1,
					"message": "获取微信用户信息失败",
					"error":   err.Error(),
				})
				return
			}

			// 如果已经执行了SQL查询（通过模板中的{{.openid}}占位符）
			// 则直接使用查询结果，否则重新执行查询
			userData := data
			// 如果SQL模板中有OpenID占位符但未能替换，这里需要重新查询
			if strings.Contains(tmpsql, "{{.openid}}") {
				// 替换模板中的{{.openid}}
				newSql := strings.ReplaceAll(tmpsql, "{{.openid}}", wxResp.OpenID)

				if newSql != tmpsql {
					// SQL发生了变化，需要重新查询
					rows, err := db.Queryx(newSql)
					CatchErr("QUERY-ERR:", err)
					if err != nil {
						c.JSON(http.StatusInternalServerError, Map{"status": 1, "message": "查询执行失败", "error": err.Error()})
						return
					}

					// 清空原来的数据，填充新数据
					userData = make([]Map, 0)
					for rows.Next() {
						mp := make(Map, 0)
						err = rows.MapScan(mp)
						if err == nil {
							// 转换值的格式
							for k, val := range mp {
								mp[k] = Conv(val)
							}
							userData = append(userData, mp)
						}
					}
				}
			}

			// 判断是否找到用户
			if len(userData) > 0 {
				// 提取用户信息
				userID := 0
				userName := ""
				if id, ok := userData[0]["UserID"]; ok {
					switch v := id.(type) {
					case float64:
						userID = int(v)
					case int:
						userID = v
					case int64:
						userID = int(v)
					case string:
						fmt.Sscanf(v, "%d", &userID)
					}
				}
				if name, ok := userData[0]["UserName"]; ok {
					if s, ok := name.(string); ok {
						userName = s
					}
				}

				// 生成JWT令牌
				token, err := GenerateToken(userID, userName)
				if err == nil {
					// 返回令牌
					c.JSON(http.StatusOK, Map{
						"status": 0,
						"token":  token,
						"openid": wxResp.OpenID,
						"data":   userData,
					})
					return
				} else {
					c.JSON(http.StatusInternalServerError, Map{
						"status":  1,
						"message": "令牌生成失败",
						"error":   err.Error(),
					})
					return
				}
			} else {
				// 未找到用户，返回openid，前端可处理注册流程
				c.JSON(http.StatusOK, Map{
					"status":  2, // 未找到用户但openid有效
					"openid":  wxResp.OpenID,
					"message": "未绑定用户",
				})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, Map{
				"status":  1,
				"message": "缺少微信授权码",
			})
			return
		}
	}

	// 处理登录请求和验证密码的逻辑
	if action == "login" && method == "POST" && len(data) > 0 {
		loginName, hasLoginName := param["loginName"]
		password, hasPassword := param["password"]

		if hasLoginName && hasPassword {
			// 判断是否找到用户
			if len(data) > 0 {
				// 从查询结果中获取密码和盐值
				dbPassword, hasDbPassword := data[0]["Password"]
				salt, hasSalt := data[0]["Salt"]

				if hasDbPassword && hasSalt {
					// 验证密码
					passwordValid := ValidatePassword(
						loginName.(string),
						password.(string),
						dbPassword.(string),
						salt.(string),
					)

					if !passwordValid {
						// 密码验证失败
						c.JSON(http.StatusUnauthorized, Map{
							"status":  1,
							"message": "用户名或密码错误",
						})
						return
					}

					// 密码验证成功，提取用户信息
					userID := 0
					userName := ""
					// 提取用户ID和名称
					if id, ok := data[0]["UserID"]; ok {
						switch v := id.(type) {
						case float64:
							userID = int(v)
						case int:
							userID = v
						case int64:
							userID = int(v)
						case string:
							fmt.Sscanf(v, "%d", &userID)
						}
					}
					if name, ok := data[0]["UserName"]; ok {
						if s, ok := name.(string); ok {
							userName = s
						}
					}

					// 生成JWT令牌
					token, err := GenerateToken(userID, userName)
					if err == nil {
						// 返回令牌
						c.JSON(http.StatusOK, Map{
							"status": 0,
							"token":  token,
							"data":   data,
						})
						return
					} else {
						c.JSON(http.StatusInternalServerError, Map{
							"status":  1,
							"message": "令牌生成失败",
							"error":   err.Error(),
						})
						return
					}
				}
			}

			// 如果代码执行到这里，说明登录失败
			c.JSON(http.StatusUnauthorized, Map{
				"status":  1,
				"message": "用户名或密码错误",
			})
			return
		}
	}

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
		log.Printf("%s %v", desc, err)
	}
}

// configureCORS 配置CORS中间件
func configureCORS() gin.HandlerFunc {
	return cors.Default()
}
