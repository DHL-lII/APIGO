package main

import (
	"bytes"
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

	"github.com/gin-contrib/cors" // 跨域资源共享中间件
	"github.com/gin-gonic/gin"    // Web框架
	"github.com/jmoiron/sqlx"     // 增强的数据库操作包

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
	// 注册通用API处理函数，支持所有HTTP方法
	r.Any(cfg.Api, Api)
	// 打印启动信息
	log.Printf("【慧工厂】·【API启动:%v】·【by 一零院长】·【2023-present】·【v250317】", cfg.Port)
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

	// 从数据库获取SQL模板
	tmpStr := ""
	CatchErr("GET-TMP:", db.Get(&tmpStr, sqlstr, action, method))
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
