// Package tracker 提供文件系统 Tracker 辅助函数
package tracker

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"gopkg.in/yaml.v3"
)

// parseFrontmatter 解析 YAML frontmatter，返回元数据 map
func parseFrontmatter(data []byte) (map[string]interface{}, error) {
	content := string(data)

	// 查找 frontmatter 边界
	start := strings.Index(content, "---\n")
	if start == -1 {
		return nil, fmt.Errorf("frontmatter start not found")
	}

	end := strings.Index(content[start+4:], "---\n")
	if end == -1 {
		return nil, fmt.Errorf("frontmatter end not found")
	}

	fmContent := content[start+4 : start+4+end]

	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		return nil, fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	return fm, nil
}

// parseFrontmatterWithContent 解析 YAML frontmatter 并返回内容和 markdown 内容
func parseFrontmatterWithContent(data []byte) (map[string]interface{}, string, error) {
	content := string(data)

	start := strings.Index(content, "---\n")
	if start == -1 {
		return nil, content, fmt.Errorf("frontmatter start not found")
	}

	end := strings.Index(content[start+4:], "---\n")
	if end == -1 {
		return nil, content, fmt.Errorf("frontmatter end not found")
	}

	fmContent := content[start+4 : start+4+end]
	markdownContent := strings.TrimSpace(content[start+4+end+4:])

	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		return nil, content, fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	return fm, markdownContent, nil
}

// formatFrontmatter 格式化 frontmatter 和内容为完整文件
func formatFrontmatter(fm map[string]interface{}, content string) []byte {
	var buf bytes.Buffer

	buf.WriteString("---\n")

	// YAML 编码
	yamlData, err := yaml.Marshal(fm)
	if err != nil {
		// 如果编码失败，手动写入必要字段
		buf.WriteString(fmt.Sprintf("id: %s\n", fm["id"]))
		buf.WriteString(fmt.Sprintf("title: %s\n", fm["title"]))
		buf.WriteString(fmt.Sprintf("status: %s\n", fm["status"]))
	} else {
		buf.Write(yamlData)
	}

	buf.WriteString("---\n")

	if content != "" {
		buf.WriteString("\n")
		buf.WriteString(content)
	}

	return buf.Bytes()
}

// parseTime 解析时间字符串
func parseTime(val interface{}) *time.Time {
	if val == nil {
		return nil
	}

	str, ok := val.(string)
	if !ok {
		return nil
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return nil
	}

	return &t
}

// parseSubTaskTitle 解析子任务标题，返回类型、编号和名称
// 例如: "P1: 需求澄清" -> ("planner", 1, "需求澄清")
func parseSubTaskTitle(title string) (string, int, string) {
	// 匹配模式: "P1: 需求澄清" 或 "G1: BDD测试脚本-v1"
	re := regexp.MustCompile(`^([PGE])(\d+):\s*(.+?)(?:-v\d+)?$`)
	matches := re.FindStringSubmatch(title)

	if len(matches) < 4 {
		return "", 0, title
	}

	typeChar := matches[1]
	num := matches[2]
	name := matches[3]

	var subTaskType string
	switch typeChar {
	case "P":
		subTaskType = "planner"
	case "G":
		subTaskType = "generator"
	case "E":
		subTaskType = "evaluator"
	}

	numInt := 0
	for _, c := range num {
		numInt = numInt*10 + int(c-'0')
	}

	return subTaskType, numInt, name
}

// parseSubTaskID 解析子任务 ID，返回父任务 ID、类型和编号
// 例如: "SYM-001-P1" -> ("SYM-001", "P", 1)
func parseSubTaskID(identifier string) (string, string, int) {
	// 匹配模式: "SYM-001-P1" 或 "SYM-001-G4"
	re := regexp.MustCompile(`^(.+)-([PGE])(\d+)$`)
	matches := re.FindStringSubmatch(identifier)

	if len(matches) < 4 {
		return identifier, "", 0
	}

	parentID := matches[1]
	typeChar := matches[2]
	numStr := matches[3]

	numInt := 0
	for _, c := range numStr {
		numInt = numInt*10 + int(c-'0')
	}

	return parentID, typeChar, numInt
}

// parseConversationHistory 解析对话历史
func parseConversationHistory(data []byte) []domain.ConversationTurn {
	content := string(data)
	var turns []domain.ConversationTurn

	// 匹配 Turn 块
	re := regexp.MustCompile(`### Turn (\d+) - (\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?)\n\n\*\*(user|assistant)\:\*\*\n\n(.+?)(?:\n---|\n##|$)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		timestamp, err := time.Parse(time.RFC3339, match[2])
		if err != nil {
			continue
		}

		turn := domain.ConversationTurn{
			Role:      match[3],
			Content:   strings.TrimSpace(match[4]),
			Timestamp: timestamp,
		}
		turns = append(turns, turn)
	}

	return turns
}

// formatConversationTurn 格式化对话回合
func formatConversationTurn(turn domain.ConversationTurn) string {
	return fmt.Sprintf("\n### Turn - %s\n\n**%s:**\n\n%s\n---\n",
		turn.Timestamp.Format(time.RFC3339),
		turn.Role,
		turn.Content,
	)
}

// statusToMark 将状态转换为标记
func statusToMark(status string) string {
	switch strings.ToLower(status) {
	case "completed", "done", "approved":
		return "✅"
	case "failed", "rejected":
		return "❌"
	case "in-progress", "pending", "waiting_review":
		return "⏳"
	default:
		return "⬜"
	}
}

// stageToStatus 将阶段状态转换为任务状态
func stageToStatus(stageStatus string) string {
	switch strings.ToLower(stageStatus) {
	case "completed":
		return "completed"
	case "in_progress":
		return "in-progress"
	case "failed":
		return "needs-attention"
	case "waiting_review":
		return "in-progress"
	default:
		return "backlog"
	}
}

// statusToStageStatus 将任务状态转换为阶段状态
func statusToStageStatus(status string) string {
	switch strings.ToLower(status) {
	case "completed", "done":
		return "completed"
	case "in-progress":
		return "in_progress"
	case "needs-attention":
		return "failed"
	case "backlog":
		return "pending"
	default:
		return "pending"
	}
}

// containsState 检查状态是否在列表中
func containsState(states []string, status string) bool {
	for _, s := range states {
		if strings.EqualFold(s, status) {
			return true
		}
	}
	return false
}

// sortByVersion 按版本号排序文件名
func sortByVersion(files []string) {
	sort.Slice(files, func(i, j int) bool {
		// 提取版本号
		vi := extractVersion(files[i])
		vj := extractVersion(files[j])
		return vi < vj
	})
}

// extractVersion 从文件名中提取版本号
func extractVersion(filename string) int {
	re := regexp.MustCompile(`-v(\d+)\.md$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return 1
	}

	v := 0
	for _, c := range matches[1] {
		v = v*10 + int(c-'0')
	}
	return v
}

// updateSubTaskStatus 更新 markdown 内容中的子任务状态
func updateSubTaskStatus(content, typeChar string, num int, name string, version int, statusMark string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// 匹配子任务行
		pattern := fmt.Sprintf("- %s%d: %s-v%d", typeChar, num, name, version)
		if strings.HasPrefix(line, pattern) {
			// 替换状态标记
			newLine := pattern + " " + statusMark
			lines[i] = newLine
		}
	}

	return strings.Join(lines, "\n")
}

// updateSubTaskStatusInContent 更新内容中的子任务状态
func updateSubTaskStatusInContent(content, typeChar string, num int, name string, version int, statusMark string) string {
	return updateSubTaskStatus(content, typeChar, num, name, version, statusMark)
}