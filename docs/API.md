# HTTP API 端点

本文档描述 Symphony 的 HTTP API 规范。

## 端点列表

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 仪表板界面 |
| `/api/v1/state` | GET | 获取当前状态 |
| `/api/v1/:identifier` | GET | 获取问题详情 |
| `/api/v1/refresh` | POST | 触发刷新 |

## 详细说明

### GET /

返回仪表板 HTML 页面，显示当前编排状态和活动问题。

### GET /api/v1/state

获取当前编排器状态。

**响应示例**:

```json
{
  "running": true,
  "current_issue": {
    "id": "123",
    "identifier": "TEST-456",
    "title": "修复登录问题",
    "state": "In Progress"
  },
  "queue_length": 5,
  "last_refresh": "2024-01-15T10:30:00Z"
}
```

### GET /api/v1/:identifier

获取指定问题的详细信息。

**路径参数**:
- `identifier`: 问题标识符（如 `TEST-456`）

**响应示例**:

```json
{
  "id": "123",
  "identifier": "TEST-456",
  "title": "修复登录问题",
  "state": "In Progress",
  "workspace_path": "/workspaces/TEST-456",
  "attempts": 2,
  "last_run": "2024-01-15T10:30:00Z"
}
```

### POST /api/v1/refresh

手动触发状态刷新。

**响应示例**:

```json
{
  "status": "refresh_triggered",
  "timestamp": "2024-01-15T10:35:00Z"
}
```