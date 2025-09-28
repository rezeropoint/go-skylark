# Go-Skylark

Go-Skylark 是一个用于对接 Skylark 低代码平台复杂 API 的 Go SDK 包。

## 简介

Skylark 是一个通过积木式搭建模式，帮助快速构建多种业务场景应用的低代码平台，支持以下模块：

- 角色管理
- 空间管理
- 标签管理
- 组织管理
- 推送包管理
- 级联选择管理
- 消息管理
- 表单管理
- 模板通知管理
- 流程管理
- 文章管理

## 功能特性

- ✅ 流程创建和管理
- ✅ 表单行创建和操作
- ✅ 字段类型支持（文本、图片、选项等）
- ✅ 七牛云图片上传
- ✅ Redis 缓存支持
- ✅ 完整的错误处理

## 安装

```bash
go get github.com/your-username/go-skylark
```

## 快速开始

```go
package main

import (
    "context"
    "github.com/your-username/go-skylark/engine"
    "github.com/your-username/go-skylark/core"
)

func main() {
    // 创建引擎配置
    config := &engine.Config{
        BaseURL: "https://your-skylark-instance.com",
        AuthHeader: "your-auth-header",
    }

    // 创建引擎实例
    skylark := engine.NewSkylarkEngine(config)

    // 创建流程
    flowData := map[string]core.TypedValue{
        "title": {Type: "string", Value: "测试流程"},
    }

    err := skylark.CreateFlow(context.Background(), "flow_id", flowData)
    if err != nil {
        panic(err)
    }
}
```

## 架构说明

### 核心模块

- **core/**: 核心数据类型和常量定义
- **engine/**: 主引擎模块，提供 SkylarkEngine 接口
- **internal/**: 内部实现模块
  - `cache/`: 缓存管理
  - `flows/`: 流程管理
  - `forms/`: 表单管理
  - `images/`: 图片处理

### 核心接口

```go
type SkylarkEngine interface {
    CreateFlow(ctx context.Context, flowKey string, data map[string]TypedValue) error
    CreateFormRow(ctx context.Context, formKey string, data map[string]TypedValue) error
}
```

### 数据类型

- **TypedValue**: 带类型的值结构
- **FieldMapping**: 字段映射配置
- **FieldOption**: 字段选项定义

## 开发

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
```

### 格式化代码

```bash
go fmt ./...
```

## 依赖

- [go-zero](https://github.com/zeromicro/go-zero) - 微服务框架
- Go 标准库

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

[MIT License](LICENSE)