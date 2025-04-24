# APIGO - SQL模板引擎API服务

<div align="center">

![APIGO Logo](https://img.shields.io/badge/APIGO-SQL%20Template%20Engine-blue)
![Go Version](https://img.shields.io/badge/Go-%3E%3D1.18-00ADD8)
![License](https://img.shields.io/badge/License-MIT-green)

</div>

APIGO是一个基于Go语言开发的轻量级API服务，支持SQL模板引擎、JWT鉴权以及微信登录功能。它能够根据数据库中定义的SQL模板动态处理API请求，无需编写额外的代码即可快速创建API接口。

## ✨ 核心特性

- **SQL模板引擎**: 通过数据库中的模板定义API，支持动态参数
- **JWT鉴权**: 内置JWT生成和验证机制，保障API安全
- **微信登录**: 支持微信小程序登录流程
- **跨域支持**: 内置CORS中间件，支持跨域请求
- **多数据库支持**: 兼容MSSQL、MySQL、PostgreSQL
- **Windows服务**: 可作为Windows服务运行

## 🚀 快速开始

### 前置条件

- Go 1.18+
- 支持的数据库: MSSQL, MySQL, PostgreSQL
- Windows系统 (作为服务运行时需要)

### 安装步骤

1. 克隆仓库:
```bash
git clone https://github.com/yourusername/apigo.git
cd apigo
```

2. 安装依赖:
```bash
go mod tidy
```

3. 编译项目:
```bash
build.bat
```

4. 配置数据库:
   - 创建必要的API表 (见下方API表结构)
   - 修改`build/m.json`配置文件

5. 运行:
```bash
start.bat
```

### 安装为Windows服务

```bash
install.bat  # 需要管理员权限
```

卸载服务:
```bash
uninstall.bat
```

## 📋 API表结构

APIGO依赖数据库中的API表来定义接口:

| 字段      | 类型           | 说明                         |
|-----------|--------------|------------------------------|
| RecordID  | nvarchar(128)| 主键                         |
| 路由      | nvarchar(128)| API路径，唯一，不支持斜杠      |
| 方法      | nvarchar(128)| HTTP方法(GET/POST/PUT/DELETE) |
| 模板      | nvarchar(MAX)| SQL语法模板                   |
| 描述      | nvarchar(128)| API接口描述                   |
| 鉴权      | int          | 0=匿名访问, 1=需要JWT认证      |
| CreateUser| int          | 创建用户ID                    |
| ReportStatus| int        | 状态标识                      |

### SQL模板示例

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
```

## ⚙️ 配置文件

`m.json` 配置项说明:

```json
{
  "driver": "mssql",              // 数据库驱动: mssql/mysql/postgres
  "dsn": "server=127.0.0.1;...",  // 数据库连接字符串
  "query": "SELECT 模板, 鉴权 FROM API WHERE 路由 =? and 方法=?", // 获取API定义的SQL
  "api": "/api/:a",               // API基础路径
  "port": 9092,                   // 服务端口
  
  "jwtSecret": "your-secret-key", // JWT签名密钥
  "jwtExpire": 7200,              // JWT过期时间(秒)
  "jwtIssuer": "apigo",           // JWT发行者
  
  "wechatAppID": "wx...",         // 微信小程序AppID
  "wechatSecret": "...",          // 微信小程序Secret
  "wechatTokenUrl": "https://api.weixin.qq.com/sns/jscode2session" // 微信接口URL
}
```

## 📱 测试页面

项目包含两个测试页面:

- `index.html`: 提供API鉴权、版本测试、令牌检测功能
- `index_wechat.html`: 用于测试微信登录流程

## 🔒 安全说明

APIGO采用特殊的密码加密方式: `md5(md5(LoginName+Password)+salt)`

用户表(`JU_User`)结构:

| 字段         | 类型           | 说明        |
|------------|--------------|------------|
| UserID     | int          | 主键        |
| UserName   | nvarchar(32) | 用户名       |
| LoginName  | nvarchar(32) | 登录名       |
| Password   | varchar(64)  | 密码        |
| Salt       | varchar(32)  | 盐值        |
| ISActive   | int          | 启用状态     |
| WeChatOpenID | varchar(128) | 微信OpenID |

## 🗂️ 项目结构

```
./src/
  ├─ m.go       → 程序主入口，JWT鉴权、API处理、微信登录
  ├─ m.json     → 配置文件
./build/        → 编译目录
  ├─ m.exe      → 编译后的可执行文件
  ├─ m.json     → 配置文件
./index.html    → 主测试页面
./index_wechat.html → 微信登录测试页面
./build.bat     → 编译脚本
./start.bat     → 开发测试启动脚本
./install.bat   → 安装脚本
./uninstall.bat → 卸载脚本
./nssm.exe      → 服务管理工具
./go.mod        → Go模块依赖配置
```

## 🤝 贡献指南

欢迎贡献代码、报告问题或提出建议。请确保遵循以下步骤:

1. Fork本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request

## 📄 许可证

本项目采用MIT许可证 - 查看 [LICENSE](LICENSE) 获取详情 