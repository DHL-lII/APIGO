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
  ├─ m.go       → 程序主入口，jwt + API鉴权模块（含ERP特殊密码验证逻辑） + 微信鉴权模块
  ├─ m.json     → 配置
./build/        → 编译目录
  ├─ m.exe        → 编译后的可执行文件
  ├─ m.json   → 配置文件
./build.bat    → 编译脚本
./nssm.exe      → 服务管理工具
./install.bat   → 安装脚本
./uninstall.bat → 卸载脚本

- **API表结构**

| 字段           | 类型           | 说明        |
| ------------ | ------------ | --------- |
| RecordID| nvarchar(128) | 主键 |
| 路由 | nvarchar(128) | 统一API接口，唯一，不支持斜杠 |
| 方法 | nvarchar(128) | GET/POST/PUT/DELETE |
| 模板 | nvarchar(128) | sql语法 |
| 描述 | nvarchar(128) | API接口描述 |
| 鉴权 | int | -- 匿名:0，鉴权：1| 
| CreateUser | int | -- 默认2| 
| ReportStatus | int | -- 默认1 | 

- **API表模板语法示例：**

```sql
-- 添加常规API查询模板（需要token鉴权）
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('1', 'users', 'GET', 'SELECT UserID, UserName, LoginName, Tel, Email FROM JU_User WHERE ISActive = 1', '获取所有用户', 1, 2, 1);

-- 添加用户登录查询模板
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('2', 'login', 'POST', 'SELECT UserID, UserName, Password, Salt FROM JU_User WHERE LoginName = ''{{.loginName}}''', '登录查询', 0, 2, 1);

-- 添加微信用户查询模板
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('3', 'wxlogin', 'POST', 'SELECT UserID, UserName FROM JU_User WHERE WeChatOpenID = ''{{.openid}}''', '微信登录查询', 0, 2, 1);

-- 添加版本信息查询API（测试NULL鉴权导致的错误）
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('4', 'ver', 'GET', 'SELECT v FROM JU_VER', '获取版本信息', NULL, 2, 1);
```
---

关于Password 加密方式：md5(md5(LoginName+Password)+salt)

-- 用户表 `JU_User`

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

#### **增强目标**

APIGO的增强目标是通过主程序逻辑、增强可维护性和提高代码质量来提升整个系统的性能和可扩展性。具体目标如下：

- **JWT集中处理**：JWT的生成、验证、过期管理：在`m.go`中处理JWT的生成、过期检查等，避免分散管理。
- **微信鉴权**：微信鉴权流程，通过微信的`code`获取`openid`，然后生成JWT。
- **API鉴权**：API特殊鉴权，通过特殊密码加密方式，（如nx：md5(md5(LoginName+Password)+salt)），认证后生成JWT。
- **质量提升**：通过代码配置和模块化，确保每个功能块单一、清晰，并易于维护。

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

- **目标**：根据认证方式分别实现微信和API鉴权，简化主程序逻辑。
- **步骤**：
  1. 实现微信鉴权逻辑：通过`code`获取`openid`，然后生成JWT。
  2. 实现API鉴权逻辑：根据md5(md5(LoginName+Password)+salt)验证API用户的身份。
  3. 测试微信和API鉴权的流程，确保认证功能独立且清晰。

##### **阶段 3：打包与部署**

- **目标**：完成APIGO的打包和部署，确保所有功能正常运行。
- **步骤**：
  1. 使用Go编译APIGO为Windows可执行文件（`.exe`）。
  2. 使用NSSM将APIGO注册为Windows服务，确保其随Windows启动。
  3. 测试APIGO的完整功能，确保所有功能正常运行。

---
##### **阶段 4：完整测试**
测试页面：index.html
标签页：API鉴权，版本测试，匿名测试，令牌检测，微信测试（移动端页面真机测试获取微信授权）

API鉴权（含特殊密码校验）
- 输入：登录名和密码
- 处理：执行SQL语句查询特定用户表，进行特殊md5(md5(LoginName+Password)+salt)验证
- 输出：返回token供前端使用
- 用途：可以对接第三方ERP系统
微信鉴权
- 输入：微信授权信息
- 处理：标准微信OAuth流程
- 输出：返回token
- 用途：支持微信用户登录，微信用户绑定
匿名测试
- 特点：无需鉴权
- 输出：版本信息
- 用途：检测匿名接口是否正常运行
令牌检测
- 输入：token
- 处理：验证token有效性
- 输出：返回users信息
- 用途：测试token功能和获取用户信息

### **总结**

该文档详细介绍了APIGO优化的各个方面，包括代码优化、JWT集成、鉴权流程的集成等。通过按照计划执行步骤，您将能够优化APIGO的整体架构，使其更加高效、可维护，并且易于扩展。