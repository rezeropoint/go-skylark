# 需要与Skylark团队沟通的接口需求

## 文档说明
本文档基于**实际API响应**的分析，列出从前端流程管理工作台角度识别出的Skylark API层面的缺失和优化建议。

**已核实实际API响应**：
- ✅ 已查看《获取某个用户处理的流程任务列表示例.md》
- ✅ 已查看《流程详情示例.md》（GET /api/v4/yaw/flows/:id）

**排除项**（已明确不需要或SDK可自行解决）：
- ❌ 批量操作接口（业务上不需要，同意时需要填写信息）
- ❌ 用户统计数据聚合接口（统计走只读数据库，不用API）
- ❌ 流程导出接口（没有这个需求）
- ✅ 列表接口缺少 flow_id/title（通过映射库+缓存解决，见《流程接口补全计划.md》）

---

## 一、需要Skylark团队支持的问题（P0）

### 1.1 催办功能缺失 ⭐⭐⭐⭐

**问题描述**

流程管理工作台的常见需求是：发起人可以催促当前处理人加快处理，但当前无催办接口。

**需求接口**

```
POST /api/v4/yaw/journeys/:journey_id/urge
```

**请求体**
```json
{
  "user_id": 123,
  "message": "请尽快处理"
}
```

**响应**
```json
{
  "success": true,
  "urged_users": [
    {
      "id": 456,
      "name": "张三"
    }
  ],
  "urged_at": "2025-01-10T10:30:00Z",
  "urge_count": 2  // 累计催办次数
}
```

**后端行为建议**
- 记录催办日志（谁在何时催办）
- 向当前处理人发送通知（站内信/推送）
- 限制催办频率（如每个流程每天只能催办1次）

**业务价值**

- 提升流程效率
- 减少线下沟通成本
- 标准化的催办流程

**优先级**：P0（常见业务需求）

---

### 1.2 标记已读接口缺失 ⭐⭐⭐

**问题描述**

抄送列表中的记录有 `read` 字段（true/false），但无接口更新此字段。用户查看抄送记录后，无法标记为已读。

**需求接口**

```
PUT /api/v4/yaw/journeys/:journey_id/assignments/:assignment_id/mark_read
```

**请求体**
```json
{
  "user_id": 123,
  "read": true
}
```

**响应**
```json
{
  "success": true
}
```

**业务价值**

- 完善抄送功能
- 用户可追踪哪些流程已查看
- 提升工作台体验

**优先级**：P1（功能完整性）

---

## 二、数据结构优化建议（P1）

### 2.1 审批历史缺少处理意见和签名 ⭐⭐⭐⭐

**问题描述**

`GET /api/v4/yaw/journeys/:journey_id/assignments` 返回每个节点的 assignment，但缺少：
- `comment`（处理意见）
- `esignature`（电子签名）
- `operation`（操作类型：approve/refuse/transfer等）

需要额外调用 `GET /api/v4/yaw/journeys/:id/moments` 获取，导致 N+1 查询问题。

**当前数据结构**
```json
{
  "id": 23637,
  "status": "processing",
  "category": "processed",
  "vertex_id": 585,
  // 缺少 comment、esignature、operation
}
```

**期望数据结构**

**方案1（推荐）**：在 assignment 中嵌套 moments
```json
{
  "id": 23637,
  "status": "processing",
  "category": "processed",
  "vertex_id": 585,
  "moments": [  // 新增：该 assignment 的所有处理记录
    {
      "id": 392,
      "status": "approved",       // 操作类型
      "comment": "同意",          // 处理意见
      "esignature": "data:image/png;base64,...",  // 电子签名
      "created_at": "2025-01-10T10:30:00Z",
      "user": {
        "id": 1,
        "name": "张三"
      }
    }
  ]
}
```

**方案2（备选）**：在 assignment 中直接增加字段（仅适用于单次处理）
```json
{
  "id": 23637,
  "status": "completed",
  "comment": "同意",       // 新增
  "esignature": "...",    // 新增
  "operation": "approved" // 新增
}
```

**业务价值**

- 流程详情页审批历史时间线只需1次请求
- 减少 N+1 查询问题
- 提升页面加载速度

**优先级**：P1（性能优化）

---

### 2.2 搜索接口支持按发起人和时间筛选 ⭐⭐⭐

**问题描述**

`POST /api/v4/yaw/flows/:id/journeys/search` 当前只支持按业务字段查询，不支持：
- 按发起人筛选（initiator_user_id）
- 按处理人筛选（processor_user_id）
- 按时间范围筛选（date_from/date_to）

**需求增强**

```json
{
  "query": {
    "8979": {
      "lft": "2022-06-19",
      "rgt": "2022-06-20"
    },
    "initiator_user_id": 123,   // 新增：发起人ID
    "processor_user_id": 456,   // 新增：处理人ID
    "date_from": "2025-01-01",  // 新增：开始日期
    "date_to": "2025-01-31"     // 新增：结束日期
  },
  "page": 1,
  "per_page": 20
}
```

**业务价值**

- 完善搜索功能
- 满足常见查询需求（如"查看张三发起的所有流程"）

**优先级**：P1（功能完整性）

---

### 2.3 列表接口支持排序 ⭐⭐⭐

**问题描述**

`GET /api/v4/yaw/flows/user_assignments.json` 不支持排序参数，无法按时间或紧急程度排序。

**需求增强**

```
GET /api/v4/yaw/flows/user_assignments.json?user_id=123&category=processed&sort_by=created_at&sort_order=desc
```

**新增参数**：
- `sort_by`: 排序字段（created_at / updated_at / current_duration_threshold）
- `sort_order`: 排序方向（asc / desc）

**业务价值**

- 用户可按自己需要排序
- 快速找到最紧急的任务

**优先级**：P2（体验优化）

---

## 三、API响应结构确认

以下问题需要确认 Skylark API 是否已支持，还是文档不完整：

### 3.1 GET /api/v4/yaw/flows/:id 返回结构确认 ✅

**已确认**：通过《流程详情示例.md》，确认接口返回**完整信息**：

✅ **Fields 包含完整配置**：
- `type`：字段类型（Field::RadioButton、Field::TextField等）
- `options`：选项列表（单选、多选字段）
- `validations`：校验规则
- `settings`：字段设置

✅ **Vertices 包含完整信息**：
- `operations`：可用操作（route、propose、approve、refuse等）
- `settings`：节点设置（assign_manually、refuse_mode等）
- `metadata.position`：节点位置坐标（x, y）
- `user_boundary`：处理人配置（cached_user_ids、organization_ids等）
- `in_edges` / `out_edges`：边信息

**结论**：此接口**数据结构完整**，无需额外优化，SDK直接封装即可。

---

### 3.2 GET /api/v4/yaw/flows/user_assignments.json 返回结构确认 ✅

**已确认**：通过《获取某个用户处理的流程任务列表示例.md》，确认接口返回：

✅ **Assignment 基本信息**：
- `id`（assignmentID）
- `journey_id`（流程记录ID）
- `vertex_id`（节点ID）
- `read`（已读状态）
- `response.user`（完整的用户信息，包含 tags、organizations）

❌ **缺失关键字段**：
- `flow_id`（流程ID）- 见问题1.1
- `flow_title`（流程名称）- 见问题1.1

✅ **Response 包含字段类型**：
- `mapped_values` 中的 `field_type`（text_field、radio_button、file等）

**结论**：主要缺失 `flow_id` 和 `flow_title`（见问题1.1）

---

## 四、错误码标准化建议

### 4.1 统一错误响应格式 ⭐⭐⭐

**问题描述**

当前接口错误响应格式不统一，前端难以判断错误类型并展示友好提示。

**建议格式**

```json
{
  "error": {
    "code": "PERMISSION_DENIED",  // 标准错误码
    "message": "您没有权限执行此操作",  // 用户友好的错误信息
    "details": "当前节点处理人为张三，您无权操作",  // 详细说明（可选）
    "timestamp": "2025-01-10T10:30:00Z"
  }
}
```

**常见错误码定义**
- `PERMISSION_DENIED` - 权限不足
- `JOURNEY_ENDED` - 流程已结束
- `JOURNEY_ABORTED` - 流程已终止
- `INVALID_OPERATION` - 无效操作
- `REQUIRED_FIELD_MISSING` - 必填字段缺失
- `VALIDATION_FAILED` - 数据校验失败
- `USER_NOT_FOUND` - 用户不存在
- `FLOW_NOT_FOUND` - 流程不存在
- `ASSIGNMENT_NOT_FOUND` - 任务不存在

**业务价值**

- 提升错误处理体验
- 减少用户困惑
- 便于前端国际化

**优先级**：P1（体验优化）

---

## 五、优先级总结

### P0（阻塞核心功能，必须沟通）

1. ⭐⭐⭐⭐ **催办功能**（问题1.1）- 常见业务需求
2. ⭐⭐⭐ **标记已读接口**（问题1.2）- 功能完整性

### P1（重要优化，影响体验）

3. ⭐⭐⭐⭐ **审批历史数据结构优化**（问题2.1）- 性能优化
4. ⭐⭐⭐ **搜索增强（发起人、时间筛选）**（问题2.2）- 功能完善
5. ⭐⭐⭐ **错误码标准化**（问题4.1）- 体验优化

### P2（体验优化，可延后）

6. ⭐⭐⭐ **列表排序支持**（问题2.3）- 便利性

---

## 六、沟通策略建议

### 第一轮沟通（本周内）

**主题**：核心缺失功能与性能优化

**重点问题**：
1. **是否支持催办功能？**（问题1.1，P0优先级）
   - 常见业务需求
   - 需要新增接口或确认是否有替代方案

2. **标记已读接口的实现方式？**（问题1.2，P0优先级）
   - assignment 有 read 字段，但无更新接口

**预期产出**：
- 确认哪些功能可以实现
- 获取预计的开发排期
- 明确技术实现方案

### 第二轮沟通（2周后）

**主题**：数据结构优化与功能完善

**重点问题**：
1. 审批历史数据结构优化方案（问题2.1）
2. 搜索功能增强（问题2.2）
3. 错误码标准化（问题4.1）

**预期产出**：
- 优化方案确认
- 功能开发排期

---

## 七、附录：快速对照表

| 需求 | 是否阻塞 | 优先级 | 预计工作量 | 影响面 |
|------|---------|--------|-----------|--------|
| 催办功能 | ✅是 | P0 | 中（新增接口+通知） | 我发起的列表 |
| 标记已读 | ✅是 | P0 | 小（简单UPDATE） | 抄送列表 |
| 审批历史优化 | ❌否 | P1 | 中（数据结构调整） | 流程详情页 |
| 搜索增强 | ❌否 | P1 | 小（查询条件扩展） | 搜索页 |
| 排序支持 | ❌否 | P2 | 小（ORDER BY） | 列表页 |
| 错误码标准化 | ❌否 | P1 | 大（全局改造） | 全局 |

---

## 八、备注

### 已排除的需求（无需沟通）

以下需求在前期分析中识别，但经业务确认**不需要实现**：

1. **批量操作接口** - 同意时需要填写信息，无法批量处理
2. **用户统计数据聚合接口** - 统计走只读数据库，不用API
3. **流程导出PDF接口** - 没有这个业务需求

### 已确认完整的接口（无需优化）

以下接口通过实际响应示例确认**数据结构完整**，SDK直接封装即可：

1. ✅ `GET /api/v4/yaw/flows/:id` - 流程详情（fields、vertices、edges完整）
2. ✅ Fields 配置完整（type、options、validations、settings）
3. ✅ Vertices 配置完整（operations、metadata.position、user_boundary）

---

**文档版本**：v2.0（基于实际API响应）
**最后更新**：2025-01-11
**下次更新计划**：与Skylark团队沟通后补充反馈结果
