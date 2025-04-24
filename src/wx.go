package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
)

// WxLogin 处理微信登录请求
func WxLogin(c *gin.Context) {
	// 解析请求参数
	param := ParseForm(c)
	code, ok := param["code"].(string)

	if !ok || code == "" {
		c.JSON(http.StatusBadRequest, Map{"status": 400, "msg": "微信code不能为空"})
		return
	}

	// 请求微信API获取openid
	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		cfg.WechatTokenUrl, cfg.WechatAppID, cfg.WechatSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "请求微信API失败"})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "读取微信API响应失败"})
		return
	}

	// 解析微信API响应
	var wxResp Map
	if err := json.Unmarshal(body, &wxResp); err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "解析微信API响应失败"})
		return
	}

	// 检查响应中是否包含错误信息
	if errcode, exists := wxResp["errcode"]; exists && errcode.(float64) != 0 {
		c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": wxResp["errmsg"]})
		return
	}

	openid := wxResp["openid"].(string)

	// 将openid添加到参数中，以供模板使用
	param["openid"] = openid

	// 获取查询模板名称，默认为"wx_login_query"
	wxLoginQueryName := "wx_login_query"
	if cfg.QueryTemplates != nil && cfg.QueryTemplates["wxLoginQuery"] != "" {
		wxLoginQueryName = cfg.QueryTemplates["wxLoginQuery"]
	}

	// 获取查询语句从数据库
	var result struct {
		模板 string `db:"模板"`
		鉴权 *int   `db:"鉴权"` // 使用指针类型，以便能够处理NULL值
	}

	if err := db.Get(&result, cfg.Query, wxLoginQueryName, "POST"); err != nil {
		c.JSON(http.StatusInternalServerError, Map{"status": 500, "msg": "获取查询模板失败"})
		return
	}

	// 查询用户信息
	var user struct {
		UserID   int    `db:"UserID"`
		UserName string `db:"UserName"`
	}

	// 使用模板处理SQL语句
	tmpl, err := template.New(wxLoginQueryName).Parse(result.模板)
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
		// 用户不存在，可以选择创建新用户或返回错误
		c.JSON(http.StatusUnauthorized, Map{"status": 401, "msg": "微信用户未绑定"})
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
			"openid":   openid,
		},
	})
}
