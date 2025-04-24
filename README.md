# APIGO 项目测试说明

## 项目介绍

APIGO是一个基于Go语言开发的API服务框架，支持通过SQL语法快速构建API接口。项目采用极简风格，具有以下特点：

- 支持JWT鉴权
- 支持ERP系统和微信登录
- 支持匿名访问和鉴权访问
- SQL模板化，灵活配置API
- 多数据库驱动支持

## 项目结构

- `src/m.go` - 主程序文件，包含JWT逻辑和API处理
- `src/nx.go` - ERP鉴权模块
- `src/wx.go` - 微信鉴权模块
- `src/m.json` - 配置文件
- `tary.go` - 任务栏工具
- `install.bat` - 安装脚本
- `uninstall.bat` - 卸载脚本
- `test.html` - 测试页面
- `build.bat` - Windows编译脚本
- `build.sh` - Linux编译脚本
- `Makefile` - Make构建文件

## 编译说明

### Windows平台编译

方法1：使用批处理脚本
```
build.bat
```

方法2：使用Make
```
make windows
```

### Linux平台编译

方法1：使用Shell脚本
```
chmod +x build.sh
./build.sh
```

方法2：使用Make
```
make linux
```

### 清理构建文件
```
make clean
```

编译完成后，所有文件将生成在`build`目录中。

## 配置数据库

在开始测试前，需要确保数据库中已添加必要的API配置：

```sql
-- 确保API表已创建，如果没有，创建它
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='API' AND xtype='U')
CREATE TABLE API (
    RecordID nvarchar(128) PRIMARY KEY,
    路由 nvarchar(128),
    方法 nvarchar(128),
    模板 nvarchar(4000),
    描述 nvarchar(128),
    鉴权 int DEFAULT 0,  -- 0=匿名访问，1=需要鉴权
    CreateUser int DEFAULT 2,
    ReportStatus int DEFAULT 1
);

-- 添加用户登录查询模板
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('1', 'login_query', 'POST', 'SELECT UserID, UserName, Password, Salt FROM JU_User WHERE LoginName = ''{{.loginName}}''', '登录查询', 0, 2, 1);

-- 添加微信用户查询模板
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('2', 'wx_login_query', 'POST', 'SELECT UserID, UserName FROM JU_User WHERE WeChatOpenID = ''{{.openid}}''', '微信登录查询', 0, 2, 1);

-- 添加匿名访问示例（产品列表）
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('3', 'products', 'GET', 'SELECT * FROM Products', '获取产品列表', 0, 2, 1);

-- 添加需要鉴权的API示例（订单列表）
INSERT INTO API (RecordID, 路由, 方法, 模板, 描述, 鉴权, CreateUser, ReportStatus)
VALUES ('4', 'orders', 'GET', 'SELECT * FROM Orders WHERE UserID = ''{{.user_id}}''', '获取用户订单', 1, 2, 1);
```

确保数据库中存在用户表和相关测试数据：

```sql
-- 创建用户表
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='JU_User' AND xtype='U')
CREATE TABLE JU_User (
    UserID int PRIMARY KEY,
    UserName nvarchar(32),
    LoginName nvarchar(32),
    Password varchar(64),
    Salt varchar(32),
    Tel varchar(64),
    Email varchar(128),
    ISActive int,
    AuthType int,
    OAuthType varchar(32),
    OAuthToken varchar(128),
    WeChatUserID varchar(128),
    WeChatOpenID varchar(128)
);

-- 添加测试用户
INSERT INTO JU_User (UserID, UserName, LoginName, Password, Salt, ISActive)
VALUES (1, '管理员', 'admin', '5f4dcc3b5aa765d61d8327deb882cf99', 'salt123', 1);

-- 创建产品表
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='Products' AND xtype='U')
CREATE TABLE Products (
    ProductID int PRIMARY KEY,
    ProductName nvarchar(100),
    Price decimal(10, 2),
    Category nvarchar(50)
);

-- 添加测试产品
INSERT INTO Products (ProductID, ProductName, Price, Category)
VALUES 
(1, '测试产品1', 99.99, '类别A'),
(2, '测试产品2', 199.99, '类别B');

-- 创建订单表
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='Orders' AND xtype='U')
CREATE TABLE Orders (
    OrderID int PRIMARY KEY,
    UserID int,
    OrderDate datetime,
    TotalAmount decimal(10, 2)
);

-- 添加测试订单
INSERT INTO Orders (OrderID, UserID, OrderDate, TotalAmount)
VALUES 
(1, 1, GETDATE(), 99.99),
(2, 1, GETDATE(), 199.99);
```

## 运行测试

1. 确保APIGO服务已启动：
   - 运行`install.bat`安装服务，或手动启动`src/m.exe`

2. 打开测试页面：
   - 在浏览器中打开`test.html`

3. 测试各功能模块：
   - **ERP登录测试**：输入用户名和密码，点击"登录"按钮
   - **微信登录测试**：输入微信code（模拟），点击"微信登录"按钮
   - **匿名API测试**：输入API路径、选择请求方法、设置参数，点击"发送请求"按钮
   - **鉴权API测试**：先登录获取token，然后输入API路径、选择请求方法、设置参数，点击"发送请求"按钮

## 注意事项

1. 测试页面中的API地址默认为`http://localhost:9091`，如需修改请在JavaScript代码中更新`API_BASE_URL`变量
2. 微信登录在本地测试环境仅为模拟测试，实际使用需要在微信环境中获取真实code
3. 测试前确保数据库连接正常，并已添加必要的测试数据
4. 默认管理员密码为纯文本"admin"的MD5值，实际项目中应使用`NxEncrypt`函数计算的值

## 故障排除

1. 如果遇到跨域问题，请确保APIGO服务正确配置了CORS
2. 如果登录失败，请检查数据库中的用户名和密码是否匹配
3. 如果API请求失败，请检查API路径是否正确，以及数据库中是否存在对应的API配置
4. 查看服务日志以获取更详细的错误信息 