package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
)

// NxLogin 处理ERP用户登录请求
func NxLogin(c *gin.Context) {
	// 解析请求参数
	param := ParseForm(c)
	loginName, ok1 := param["loginName"].(string)
	password, ok2 := param["password"].(string)

	if !ok1 || !ok2 || loginName == "" || password == "" {
		c.JSON(http.StatusBadRequest, Map{"status": 400, "msg": "用户名或密码不能为空"})
		return
	}

	// 获取查询模板名称，默认为"login_query"
	loginQueryName := "login_query"
	if cfg.QueryTemplates != nil && cfg.QueryTemplates["loginQuery"] != "" {
		loginQueryName = cfg.QueryTemplates["loginQuery"]
	}

	// 获取用户SQL查询语句从数据库
	var result struct {
		模板 string `db:"模板"`
		鉴权 *int   `db:"鉴权"` // 使用指针类型，以便能够处理NULL值
	}

	if err := db.Get(&result, cfg.Query, loginQueryName, "POST"); err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "获取查询模板失败"})
		return
	}

	// 查询用户信息
	var user struct {
		UserID   int    `db:"UserID"`
		UserName string `db:"UserName"`
		Password string `db:"Password"`
		Salt     string `db:"Salt"`
	}

	// 使用模板处理SQL语句
	tmpl, err := template.New(loginQueryName).Parse(result.模板)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "解析SQL模板失败"})
		return
	}

	// 执行模板
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, param); err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "执行SQL模板失败"})
		return
	}

	sqlQuery := buf.String()

	// 查询用户
	if err := db.Get(&user, sqlQuery); err != nil {
		c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": "用户不存在"})
		return
	}

	// 验证密码 - md5(md5(LoginName+Password)+salt)
	passwordHash := NxEncrypt(loginName, password, user.Salt)
	if passwordHash != user.Password {
		c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": "密码错误"})
		return
	}

	// 生成JWT令牌
	token, err := GenerateToken(user.UserID, user.UserName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "生成令牌失败"})
		return
	}

	// 返回令牌
	c.JSON(http.StatusOK, Map{
		"status": 0,
		"data": Map{
			"token":    token,
			"user_id":  user.UserID,
			"username": user.UserName,
			"expire":   cfg.JwtExpire,
		},
	})
}

// NxEncrypt 使用nx的加密方式加密密码
// 加密方式：md5(md5(LoginName+Password)+salt)
func NxEncrypt(loginName, password, salt string) string {
	// 第一次MD5: md5(LoginName+Password)
	firstMD5 := md5Sum(loginName + password)
	// 第二次MD5: md5(firstMD5+salt)
	secondMD5 := md5Sum(firstMD5 + salt)
	return secondMD5
}

// md5Sum 计算字符串的MD5哈希值
func md5Sum(text string) string {
	hash := md5.New()
	hash.Write([]byte(text))
	return hex.EncodeToString(hash.Sum(nil))
}
