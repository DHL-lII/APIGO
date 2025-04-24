# APIGO优化文档（最终版）

## **重要事情说三遍**

不允许违背了Go语言简洁明了的设计理念。
不允许违背了Go语言简洁明了的设计理念。
不允许违背了Go语言简洁明了的设计理念。

不允许存在硬编码逻辑，保持程序通用性，配置；且文件目录保持扁平
不允许存在硬编码逻辑，保持程序通用性，配置；且文件目录保持扁平
不允许存在硬编码逻辑，保持程序通用性，配置；且文件目录保持扁平

关于源码质量说明：

- 所有源码参考m.go风格编写，保持极简风格
- 除注释，格式，以外尽可能压缩代码，提高代码质量
- 项目的文件结构应当保持扁平，不允许创建多层嵌套的目录结构。

### **项目背景**

./src/m.go -- 初版

最初版：支持sql语法的api服务端
最终版：待实现，基于最初版，根据文档描述开发最终版本的 APIGO服务端程序

目录结构：

./src/
  ├─ m.go       → 程序主入口，含jwt完整逻辑
  ├─ m.json     → 配置
  ├─ nx.go      → ERP 鉴权模块
  ├─ wx.go      → 微信鉴权模块
  ├─ m.exe      → 主程序编译
./tary.go       → 任务栏图标(独立运行)
./nssm.exe      → 服务管理工具
./install.bat   → 安装脚本(独立运行)
./uninstall.bat → 卸载脚本(独立运行)

- **API表结构**

| 字段           | 类型           | 说明        |
| ------------ | ------------ | --------- |
| RecordID| nvarchar(128) | 主键 |
| 路由 | nvarchar(128) | API接口，不支持A/B |
| 方法 | nvarchar(128) | GET/POST/PUT/DELETE |
| 模板 | nvarchar(128) | sql语法 |
| 描述 | nvarchar(128) | API接口描述 |
| 鉴权 | int | -- 匿名:0，鉴权：1| 
| CreateUser | int | -- 默认2| 
| ReportStatus | int | -- 默认1 | 

- **API表模板语法示例：**

```sql
INSERT INTO AUTH (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus) VALUES 
(1, 'login', 'POST', 'SELECT UserID, UserName AS Name, Password, Salt FROM JU_User WHERE LoginName=''{{.loginName}}''', '登录鉴权', 0, 2, 1);
```
---

关于Password 加密方式：md5(md5(LoginName+Password)+salt)

-- 用户表 `JU_User`（nxerp）

| 字段           | 类型           | 说明        |
| ------------ | ------------ | --------- |
| UserID       | int          | 主键        |
| UserName     | nvarchar(32) | 用户名       |
| LoginName    | nvarchar(32) | 登录名       |
| Password     | varchar(64)  | 密码        |
| Salt         | varchar(32)  | 盐值        |
| Tel          | varchar(64)  | 电话        |
| Email        | varchar(128) | 邮箱        |
| ISActive     | int          | 启用状态      |
| AuthType     | int          | 鉴权类型      |
| OAuthType    | varchar(32)  | OAuth类型   |
| OAuthToken   | varchar(128) | Token     |
| WeChatUserID | varchar(128) | 微信 UID    |
| WeChatOpenID | varchar(128) | 微信 OpenID |

- **配置文件示例**

```JSON
{
  "driver": "mssql",
  "dsn": "server=127.0.0.1;user id=sa;password=Fb2233;database=g4;port=1433;encrypt=disable",
  "query": "SELECT 模板 FROM API WHERE 路由 = ? AND 方法 = ?",
  "api": "/api/:a",
  "port": 9092,
  "memo": "驱动可选：mssql/mysql/postgres",

  "jwtSecret": "Fb2233",
  "jwtExpire": 7200,
  "jwtIssuer": "apigo",

  "wechatAppID": "wx6666666666666666",
  "wechatSecret": "66666666666666666666666666666666",
  "wechatTokenUrl": "https://api.weixin.qq.com/sns/jscode2session",

}
```
---

#### **优化目标**

APIGO的优化目标是通过主程序逻辑、增强可维护性和提高代码质量来提升整个系统的性能和可扩展性。具体优化目标如下：

1. **集成JWT逻辑到`m.go`**：JWT的生成、验证、过期管理等逻辑直接由主程序（`m.go`）管控，避免冗余代码。
2. **简化鉴权流程**：微信认证和ERP集成认证分开处理，确保清晰的流程管理。
3. **提升代码质量**：通过代码配置和模块化，确保每个功能块单一、清晰，并易于维护。

---

- **步骤总结**

1. 集成JWT逻辑到`m.go`

- **集中处理JWT生成、验证、过期管理**：在`m.go`中处理JWT的生成、过期检查等，避免分散管理。

2. 简化鉴权流程

- **微信鉴权**：独立的微信鉴权流程，通过微信的`code`获取`openid`，然后生成JWT。
- **ERP集成鉴权**：每个ERP系统使用不同的密码加密方式，（如nx：md5(md5(LoginName+Password)+salt)），认证后生成JWT。

---

### **1.3 APIGO任务栏图标**

任务栏图标功能将通过以下步骤实现，提供图标显示、右键菜单和服务管理功能：

1. **使用`systary`库创建任务栏图标**：通过`systray`库将APIGO以任务栏图标的形式呈现，并提供右键菜单。
2. **右键菜单**：提供启动、停止服务、查看配置、查看日志等选项。
3. **服务管理**：通过NSSM将APIGO注册为Windows服务，允许通过任务栏菜单进行启动、停止操作。

---

#### **1.4 任务栏图标功能具体设计**

1. **初始化任务栏图标**：使用`systray.SetIcon`方法设置图标，`systray.SetTooltip`设置提示。
2. **右键菜单**：创建“启动服务”、“停止服务”、“查看配置”、“查看日志”等菜单项，并绑定处理逻辑。
3. **服务启动和停止**：通过NSSM启动或停止APIGO服务，保证APIGO后台运行。
4. **查看配置和日志**：提供打开配置文件和日志文件的选项。

### **1.5 代码示例：任务栏图标**

```go
package main

import (
    "fmt"
    "log"
    "os/exec"
    "github.com/getlantern/systray"
)

func main() {
    systray.Run(onReady, onExit)
}

func onReady() {
    systray.SetIcon([]byte{ /* 二进制图标数据 */ })
    systray.SetTooltip("APIGO 服务管理")

    mStart := systray.AddMenuItem("启动服务", "启动APIGO服务")
    mStop := systray.AddMenuItem("停止服务", "停止APIGO服务")
    mConfig := systray.AddMenuItem("配置", "打开配置文件")
    mQuit := systray.AddMenuItem("退出", "退出程序")

    go func() {
        <-mStart.ClickedCh
        startService()
    }()

    go func() {
        <-mStop.ClickedCh
        stopService()
    }()

    go func() {
        <-mConfig.ClickedCh
        openConfig()
    }()

    go func() {
        <-mQuit.ClickedCh
        systray.Quit()
    }()
}

func startService() {
    cmd := exec.Command("nssm", "start", "APIGO")
    err := cmd.Run()
    if err != nil {
        log.Println("启动服务失败:", err)
    } else {
        fmt.Println("APIGO服务已启动")
    }
}

func stopService() {
    cmd := exec.Command("nssm", "stop", "APIGO")
    err := cmd.Run()
    if err != nil {
        log.Println("停止服务失败:", err)
    } else {
        fmt.Println("APIGO服务已停止")
    }
}

func openConfig() {
    cmd := exec.Command("notepad", "apigo.json")
    err := cmd.Start()
    if err != nil {
        log.Println("打开配置文件失败:", err)
    }
}

func onExit() {
    // 清理任务栏图标
}
```

---

### **2. 完整的计划文档**

#### **目标**：

- **优化代码质量**：减少冗余、提升可维护性。
- **简化认证流程**：集成JWT逻辑，并将微信、ERP集成认证分别管理。
- **任务栏图标功能**：提供服务管理和配置管理功能。

---

#### **计划步骤：**

##### **阶段 1：集成JWT逻辑到`m.go`**

- **目标**：将JWT生成、验证和过期管理逻辑集成到`m.go`中。
- **步骤**：
  1. 在`m.go`中定义JWT密钥。
  2. 实现JWT生成和验证逻辑，确保每个API请求都能正确通过JWT验证。
  3. 实现JWT过期管理，确保JWT在过期后无法继续使用。
  4. 匿名鉴权除外
  5. 测试JWT的生成、验证和过期机制，确保JWT认证的正确性。

##### **阶段 2：简化鉴权流程**

- **目标**：根据认证方式分别实现微信和ERP鉴权，简化主程序逻辑。
- **步骤**：
  1. 实现微信鉴权逻辑：通过`code`获取`openid`，然后生成JWT。
  2. 实现ERP鉴权逻辑：根据md5(md5(LoginName+Password)+salt)验证ERP用户的身份。
  3. 测试微信和ERP鉴权的流程，确保认证功能独立且清晰。

##### **阶段 3：实现任务栏图标与服务管理**

- **目标**：实现APIGO的任务栏图标功能，提供启动/停止服务、查看配置等功能。
- **步骤**：
  1. 使用`systray`库创建任务栏图标。
  2. 创建右键菜单，提供启动、停止服务、查看配置文件和查看日志的选项。
  3. 使用NSSM注册APIGO为Windows服务，确保APIGO后台运行。
  4. 测试任务栏图标功能，确保服务管理和配置文件查看等功能正常工作。

##### **阶段 4：打包与部署**

- **目标**：完成APIGO的打包和部署，确保所有功能正常运行。
- **步骤**：
  1. 使用Go编译APIGO为Windows可执行文件（`.exe`）。
  2. 使用NSSM将APIGO注册为Windows服务，确保其随Windows启动。
  3. NSSM与任务栏运行分别独立，没有NSSM 任务栏也可长期驻留运行
  4. 测试APIGO的完整功能，确保所有功能正常运行。

---
### **测试**
测试页面：，API鉴权，erp鉴权，微信鉴权，版本测试，匿名测试，令牌测试

API鉴权
- 输入：用户名和密码
- 处理：标准API鉴权流程
- 输出：返回token
- 用途：常规API访问授权
ERP鉴权
- 输入：登录名和密码
- 处理：执行SQL语句查询特定用户表，进行特殊md5(md5(LoginName+Password)+salt)验证
- 输出：返回token供前端使用
- 用途：与现有ERP系统集成的鉴权方式
微信鉴权
- 输入：微信授权信息
- 处理：标准微信OAuth流程
- 输出：返回token
- 用途：支持微信用户登录
版本测试
- 问题：当前会报错
- 原因：版本语句中鉴权字段=null
- 错误：scannable dest type struct with >1 columns (2) in result
- 解决方向：需修复版本查询SQL结构与接收结构的不匹配问题
匿名测试
- 接口：/api2/ver
- 特点：无需鉴权
- 输出：版本信息
- 用途：检查系统版本信息
令牌测试
- 输入：token
- 处理：验证token有效性
- 输出：返回users信息
- 用途：测试token功能和获取用户信息

### **总结**

该文档详细介绍了APIGO优化的各个方面，包括代码优化、JWT集成、鉴权流程的简化、任务栏图标的实现等。通过按照计划执行步骤，您将能够优化APIGO的整体架构，使其更加高效、可维护，并且易于扩展。同时，任务栏图标和服务管理功能为系统的管理和使用提供了便利。