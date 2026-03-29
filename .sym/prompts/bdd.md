# BDD 规则生成提示模板

你是一个 BDD（行为驱动开发）专家。请根据以下需求信息生成 Gherkin 格式的 BDD 规则。

## 需求标题
{{ issue.title }}

## 需求描述
{{ issue.description }}

## 澄清历史
{{ clarification_history }}

请生成符合以下格式的 BDD 规则：

```gherkin
Feature: [功能名称]

Scenario: [场景名称]
  Given [前置条件]
  When [触发动作]
  Then [预期结果]
```

## 生成要求

1. **Feature 描述**
   - 使用简洁明了的功能名称
   - 与需求标题保持一致

2. **Scenario 设计**
   - 每个场景应该覆盖一个具体的使用情况
   - 包含正向场景（成功路径）和负向场景（失败/异常路径）
   - 场景名称应清晰描述场景意图

3. **Given-When-Then 结构**
   - Given: 描述前置条件和初始状态
   - When: 描述用户行为或系统事件
   - Then: 描述预期结果或系统响应
   - 可以使用 And 连接多个条件或结果

4. **场景覆盖**
   - 正常流程场景
   - 边界条件场景
   - 异常处理场景
   - 错误处理场景

请以 JSON 格式返回：
```json
{
  "feature": {
    "name": "功能名称",
    "description": "功能描述"
  },
  "scenarios": [
    {
      "name": "场景名称",
      "given": ["前置条件1", "前置条件2"],
      "when": ["触发动作"],
      "then": ["预期结果1", "预期结果2"],
      "tags": ["@tag1", "@tag2"]
    }
  ],
  "summary": "BDD 规则摘要"
}
```

## 示例

**输入需求：**
用户登录功能

**输出 BDD 规则：**
```gherkin
Feature: 用户登录功能

  @happy_path
  Scenario: 用户使用邮箱登录成功
    Given 用户在登录页面
    When 用户输入有效的邮箱 "test@example.com"
    And 用户输入正确的密码 "password123"
    And 用户点击登录按钮
    Then 用户应该被重定向到首页
    And 用户应该看到欢迎消息

  @negative
  Scenario: 用户使用无效邮箱登录失败
    Given 用户在登录页面
    When 用户输入无效的邮箱 "invalid-email"
    And 用户点击登录按钮
    Then 用户应该看到错误消息 "邮箱格式不正确"

  @negative
  Scenario: 用户使用错误密码登录失败
    Given 用户在登录页面
    When 用户输入有效的邮箱 "test@example.com"
    And 用户输入错误的密码 "wrongpassword"
    And 用户点击登录按钮
    Then 用户应该看到错误消息 "密码错误"

  @edge_case
  Scenario: 用户连续多次登录失败后被锁定
    Given 用户在登录页面
    And 用户已连续登录失败 5 次
    When 用户再次尝试登录
    Then 用户应该看到错误消息 "账户已被临时锁定"
    And 用户应该被提示等待 30 分钟后重试
```