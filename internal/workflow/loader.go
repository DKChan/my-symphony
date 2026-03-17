// Package workflow 提供WORKFLOW.md文件的加载和解析功能
package workflow

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dministrator/symphony/internal/config"
	"gopkg.in/yaml.v3"
)

var (
	// ErrMissingWorkflowFile 工作流文件缺失错误
	ErrMissingWorkflowFile = errors.New("missing_workflow_file")
	// ErrWorkflowParseError 工作流解析错误
	ErrWorkflowParseError = errors.New("workflow_parse_error")
	// ErrWorkflowFrontMatterNotMap 前置内容不是映射错误
	ErrWorkflowFrontMatterNotMap = errors.New("workflow_front_matter_not_a_map")
)

// Definition 工作流定义
type Definition struct {
	// Config 配置映射
	Config map[string]any `json:"config"`
	// PromptTemplate 提示模板
	PromptTemplate string `json:"prompt_template"`
}

// Loader 工作流加载器
type Loader struct {
	path string
}

// NewLoader 创建新的工作流加载器
func NewLoader(path string) *Loader {
	return &Loader{path: path}
}

// Load 加载工作流文件
func (l *Loader) Load() (*Definition, error) {
	content, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrMissingWorkflowFile, l.path)
		}
		return nil, fmt.Errorf("%w: %v", ErrWorkflowParseError, err)
	}

	return Parse(content)
}

// Parse 解析工作流内容
func Parse(content []byte) (*Definition, error) {
	// 检查是否有YAML前置内容
	if !bytes.HasPrefix(content, []byte("---\n")) {
		// 没有前置内容，整个文件作为提示模板
		return &Definition{
			Config:         make(map[string]any),
			PromptTemplate: strings.TrimSpace(string(content)),
		}, nil
	}

	// 查找前置内容结束标记
	endIndex := bytes.Index(content[4:], []byte("\n---"))
	if endIndex == -1 {
		return nil, fmt.Errorf("%w: no closing --- found", ErrWorkflowParseError)
	}

	frontMatter := content[4 : endIndex+4]
	body := content[endIndex+8:]

	// 解析YAML前置内容
	var rawConfig any
	if err := yaml.Unmarshal(frontMatter, &rawConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWorkflowParseError, err)
	}

	// 验证配置是映射类型
	configMap, ok := rawConfig.(map[string]any)
	if !ok {
		return nil, ErrWorkflowFrontMatterNotMap
	}

	return &Definition{
		Config:         configMap,
		PromptTemplate: strings.TrimSpace(string(body)),
	}, nil
}

// GetPath 获取工作流文件路径
func (l *Loader) GetPath() string {
	return l.path
}

// ParseConfig 从工作流定义解析配置
func (d *Definition) ParseConfig() (*config.Config, error) {
	return config.ParseConfig(d.Config)
}