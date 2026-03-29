// Package bdd 提供 BDD Gherkin 解析和渲染功能
package bdd

import (
	"regexp"
	"strings"
)

// Feature 表示 Gherkin Feature 结构
type Feature struct {
	Name        string     `json:"name"`         // Feature 名称
	Description string     `json:"description"`  // Feature 描述
	Scenarios   []Scenario `json:"scenarios"`    // 场景列表
}

// Scenario 表示 Gherkin Scenario 结构
type Scenario struct {
	Name  string   `json:"name"`  // Scenario 名称
	Steps []Step   `json:"steps"` // 步骤列表
	Tags  []string `json:"tags"`  // 标签列表
}

// Step 表示 Gherkin 步骤
type Step struct {
	Keyword string `json:"keyword"` // 关键词：Given, When, Then, And, But
	Text    string `json:"text"`    // 步骤文本
}

// ParseGherkin 解析 Gherkin 内容，返回结构化的 Feature 数据
func ParseGherkin(content string) (*Feature, error) {
	if content == "" {
		return nil, nil
	}

	feature := &Feature{
		Scenarios: []Scenario{},
	}

	lines := strings.Split(content, "\n")
	currentScenario := -1
	inScenario := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过空行和注释
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 解析 Feature 行
		if strings.HasPrefix(trimmed, "Feature:") || strings.HasPrefix(trimmed, "功能:") {
			feature.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "Feature:"))
			feature.Name = strings.TrimSpace(strings.TrimPrefix(feature.Name, "功能:"))
			// 描述从 Feature 行之后开始，直到第一个 Scenario
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" || strings.HasPrefix(nextLine, "#") {
					continue
				}
				if strings.HasPrefix(nextLine, "Scenario") || strings.HasPrefix(nextLine, "场景") {
					break
				}
				if feature.Description != "" {
					feature.Description += "\n"
				}
				feature.Description += nextLine
			}
			continue
		}

		// 解析 Scenario 行
		if strings.HasPrefix(trimmed, "Scenario:") || strings.HasPrefix(trimmed, "场景:") {
			scenarioName := strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario:"))
			scenarioName = strings.TrimSpace(strings.TrimPrefix(scenarioName, "场景:"))
			scenario := Scenario{
				Name:  scenarioName,
				Steps: []Step{},
				Tags:  []string{},
			}
			feature.Scenarios = append(feature.Scenarios, scenario)
			currentScenario = len(feature.Scenarios) - 1
			inScenario = true
			continue
		}

		// 解析 Scenario Outline
		if strings.HasPrefix(trimmed, "Scenario Outline:") || strings.HasPrefix(trimmed, "场景大纲:") {
			scenarioName := strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario Outline:"))
			scenarioName = strings.TrimSpace(strings.TrimPrefix(scenarioName, "场景大纲:"))
			scenario := Scenario{
				Name:  scenarioName,
				Steps: []Step{},
				Tags:  []string{},
			}
			feature.Scenarios = append(feature.Scenarios, scenario)
			currentScenario = len(feature.Scenarios) - 1
			inScenario = true
			continue
		}

		// 解析 Tags
		if strings.HasPrefix(trimmed, "@") && !inScenario {
			// Feature 级别的 Tags，暂时忽略
			continue
		}
		if strings.HasPrefix(trimmed, "@") && inScenario && currentScenario >= 0 {
			tags := parseTags(trimmed)
			feature.Scenarios[currentScenario].Tags = append(feature.Scenarios[currentScenario].Tags, tags...)
			continue
		}

		// 解析步骤
		if inScenario && currentScenario >= 0 {
			step := parseStep(trimmed)
			if step != nil {
				feature.Scenarios[currentScenario].Steps = append(feature.Scenarios[currentScenario].Steps, *step)
			}
		}
	}

	// 如果没有解析到 Feature 名称，尝试从内容推断
	if feature.Name == "" && content != "" {
		feature.Name = "未命名 Feature"
	}

	return feature, nil
}

// parseStep 解析单个步骤
func parseStep(line string) *Step {
	keywords := []string{"Given", "When", "Then", "And", "But", "假如", "当", "那么", "并且", "但是"}

	for _, keyword := range keywords {
		if strings.HasPrefix(line, keyword+":") || strings.HasPrefix(line, keyword+" ") {
			text := strings.TrimSpace(strings.TrimPrefix(line, keyword+":"))
			text = strings.TrimSpace(strings.TrimPrefix(text, keyword+" "))
			return &Step{
				Keyword: keyword,
				Text:    text,
			}
		}
	}

	// 如果没有明确的关键词，可能是 Examples 的一部分或其他内容
	return nil
}

// parseTags 解析标签行
func parseTags(line string) []string {
	// 移除 @ 前缀并按空格分割
	line = strings.TrimPrefix(line, "@")
	tags := strings.Fields(line)
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if strings.HasPrefix(tag, "@") {
			result = append(result, strings.TrimPrefix(tag, "@"))
		} else {
			result = append(result, tag)
		}
	}
	return result
}

// StepKeywordCSSClass 根据步骤关键词返回 CSS 类名
func StepKeywordCSSClass(keyword string) string {
	switch keyword {
	case "Given", "假如":
		return "step-given"
	case "When", "当":
		return "step-when"
	case "Then", "那么":
		return "step-then"
	case "And", "并且":
		return "step-and"
	case "But", "但是":
		return "step-but"
	default:
		return "step-other"
	}
}

// ScenarioCount 返回 Feature 中的场景数量
func (f *Feature) ScenarioCount() int {
	return len(f.Scenarios)
}

// TotalSteps 返回 Feature 中的总步骤数
func (f *Feature) TotalSteps() int {
	total := 0
	for _, s := range f.Scenarios {
		total += len(s.Steps)
	}
	return total
}

// ValidateBDDContent 验证 BDD 内容是否有效
func ValidateBDDContent(content string) bool {
	if content == "" {
		return false
	}

	// 必须包含 Feature 关键词
	hasFeature := strings.Contains(content, "Feature:") || strings.Contains(content, "功能:")

	// 必须包含至少一个 Scenario
	hasScenario := strings.Contains(content, "Scenario:") ||
		strings.Contains(content, "场景:") ||
		strings.Contains(content, "Scenario Outline:") ||
		strings.Contains(content, "场景大纲:")

	return hasFeature && hasScenario
}

// ExtractFeatureName 从内容中提取 Feature 名称
func ExtractFeatureName(content string) string {
	re := regexp.MustCompile(`(?:Feature|功能):\s*(.+?)(?:\n|$)`)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// FormatGherkinForDisplay 格式化 Gherkin 内容用于显示（添加高亮标记）
func FormatGherkinForDisplay(content string) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 高亮 Feature 行
		if strings.HasPrefix(trimmed, "Feature:") || strings.HasPrefix(trimmed, "功能:") {
			result = append(result, `<span class="gherkin-feature">`+line+"</span>")
			continue
		}

		// 高亮 Scenario 行
		if strings.HasPrefix(trimmed, "Scenario:") || strings.HasPrefix(trimmed, "场景:") ||
			strings.HasPrefix(trimmed, "Scenario Outline:") || strings.HasPrefix(trimmed, "场景大纲:") {
			result = append(result, `<span class="gherkin-scenario">`+line+"</span>")
			continue
		}

		// 高亮步骤
		step := parseStep(trimmed)
		if step != nil {
			cssClass := StepKeywordCSSClass(step.Keyword)
			result = append(result, `<span class="gherkin-step `+cssClass+`">`+line+"</span>")
			continue
		}

		// 高亮 Tags
		if strings.HasPrefix(trimmed, "@") {
			result = append(result, `<span class="gherkin-tags">`+line+"</span>")
			continue
		}

		// 其他内容
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}