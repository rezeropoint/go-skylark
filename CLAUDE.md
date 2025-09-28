# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

go-skylark 是一个用于对接 Skylark 低代码平台复杂 API 的 Go SDK 包。Skylark 是一个通过积木式搭建模式，帮助快速构建多种业务场景应用的低代码平台，支持角色管理、空间管理、标签管理、组织管理、推送包管理、级联选择管理、消息管理、表单管理、模板通知管理、流程管理、文章管理等模块。

## 代码架构

### 核心模块结构

- **core/**: 核心数据类型和常量定义
  - `auth.go`: 认证相关结构体
  - `field.go`: 字段类型定义，包括 TypedValue、FieldMapping、FieldOption 等
  - `var.go`: 错误定义、API 路径常量、字段类型常量等
  - `utils.go`: 工具函数
  - `address.go`: 地址相关功能
  - `cache.go`: 缓存接口定义

- **engine/**: 主引擎模块
  - `engine.go`: 定义 SkylarkEngine 接口，提供 CreateFlow 和 CreateFormRow 方法
  - `config.go`: 引擎配置结构
  - `handler.go`: 具体实现逻辑

- **internal/**: 内部实现模块
  - `cache/`: 缓存管理，实现 CacheInterface 接口
  - `flows/`: 流程管理，实现 SkylarkFlowRegistry 接口，依赖 CacheInterface
  - `forms/`: 表单管理，实现 SkylarkFormRegistry 接口，依赖 CacheInterface
  - `images/`: 图片处理功能

### 核心接口

1. **SkylarkEngine**: 主引擎接口
   - `CreateFlow()`: 创建并启动流程
   - `CreateFormRow()`: 创建表单行

2. **SkylarkFlowRegistry**: 流程注册表接口
3. **SkylarkFormRegistry**: 表单注册表接口
4. **CacheInterface**: 缓存操作接口，定义字段映射缓存方法

### 数据类型系统

- **TypedValue**: 带类型的值结构，支持 string、imageURL、imageBase64 等类型
- **FieldMapping**: 字段映射，包含字段 ID、标识键、类型和选项
- **FieldOption**: 字段选项，支持单选、多选、下拉等选择类型

## 常用开发命令

### 构建和测试
```bash
# 构建模块
go build ./...

# 运行测试
go test ./...

# 运行特定包的测试
go test ./core
go test ./engine
go test ./internal/flows

# 格式化代码
go fmt ./...

# 运行静态分析
go vet ./...
```

### 模块管理
```bash
# 整理依赖
go mod tidy

# 验证依赖
go mod verify

# 查看依赖图
go mod graph
```

## 重要约定

### 错误处理
- 使用 `core/var.go` 中预定义的错误类型
- 数据库操作时使用 sql.NullString 或 sql.NullTime 防止 NULL 值转换错误

### 字段类型处理
- 图片字段使用 `_Img` 后缀
- Base64 图片字段使用 `_Base64Img` 后缀
- 选项类型字段包括: RadioButton、Checkbox、SelectField、MultipleSelectField

### API 路径
- 流程 API: `/api/v4/yaw/flows/`
- 表单 API: `/api/v4/forms/`
- 附件 API: `/api/v4/attachments/uptoken`

## 依赖关系

主要依赖:
- **go-zero**: 微服务框架，用于 Redis 客户端等
- **标准库**: context、errors 等

## 开发注意事项

1. 所有外部 API 调用都需要传入认证头 (authHeader)
2. 支持七牛云图片上传，使用固定的 x:key 值 "1593586993541"
3. 使用 Redis 进行缓存管理，通过 CacheInterface 接口抽象缓存操作
4. 字段值类型需要通过 TypedValue 结构进行包装
5. 选项字段需要通过 IsOptionField() 函数判断类型