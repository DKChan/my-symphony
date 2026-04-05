package components

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
)

// TemplateEngine 模板渲染引擎
type TemplateEngine struct {
	templates *template.Template
	fs        embed.FS
}

// StatusBadgeData 状态徽章数据
type StatusBadgeData struct {
	Class    string
	ID       string
	Text     string
	ShowDot  bool
}

// TemplateData 模板数据结构
type TemplateData struct {
	// 公共数据
	Title       string
	TitleStyle  string
	PageName    string
	HeroCopy    string
	NeedsHTMX   bool
	NeedsSSE    bool
	ShowBackButton bool
	BackURL     string
	BackText    string
	StatusBadge *StatusBadgeData
	ActionButtonsHTML template.HTML

	// 页面特定数据
	Issue       *domain.Issue
	StageState  *domain.StageState
	State       *domain.OrchestratorState
	Conversation []domain.ConversationTurn
	Report      *domain.VerificationReport
	BDDContent  string
	ArchContent string
	TDDContent  string

	// 计算后的辅助数据
	ElapsedDisplay     string
	StageDisplay       string
	StatusDisplay      string
	StateClass         string
	IsWaitingForAnswer bool
	IsImplementation   bool
	IsNeedsAttention   bool
	IsVerification     bool
	CurrentQuestion    string
	RoundProgress      string
	CurrentRound       int
	FailedAtStr        string

	// 仪表板特定数据
	Now             time.Time
	MetricRunning   int
	MetricRetrying  int
	MetricTokens    string
	MetricRuntime   string
	RateLimitsHTML  template.HTML
	KanbanHTML      template.HTML
	FilterBarHTML   template.HTML
	TaskListHTML    template.HTML
	ConversationHTML template.HTML
	BDDContentHTML  template.HTML
	ArchContentHTML template.HTML
	TDDContentHTML  template.HTML

	// 错误页面数据
	ErrorTitle   string
	ErrorMessage string

	// 任务创建数据
	ParentInfo   *ParentTaskInfo
	SubTasks     []SubTaskInfo
}

// ParentTaskInfo 父任务信息
type ParentTaskInfo struct {
	Identifier string
	Title      string
}

// SubTaskInfo 子任务信息
type SubTaskInfo struct {
	Identifier string
	Title      string
}

// NewTemplateEngine 创建模板引擎实例
func NewTemplateEngine(templateFS embed.FS) (*TemplateEngine, error) {
	// 使用 ParseFS 正确加载模板，避免 define block 被覆盖问题
	tmpl, err := template.New("layout.html").
		Funcs(template.FuncMap{
			"stageDisplay":     getStageDisplay,
			"statusDisplay":    getStatusDisplay,
			"formatDuration":   formatDurationForDetail,
			"formatInt":        common.FormatInt,
			"prettyValue":      common.PrettyValue,
			"stateBadgeClass":  common.StateBadgeClass,
			"lower":            strings.ToLower,
			"upper":            strings.ToUpper,
			"trim":             strings.TrimSpace,
			"safeHTML":         func(s string) template.HTML { return template.HTML(s) },
			"safeJS":           func(s string) template.JS { return template.JS(s) },
			"add":              func(a, b int) int { return a + b },
			"sub":              func(a, b int) int { return a - b },
			"mul":              func(a, b int) int { return a * b },
			"div":              func(a, b int) int { if b == 0 { return 0 }; return a / b },
			"eq":               func(a, b string) bool { return a == b },
			"ne":               func(a, b string) bool { return a != b },
			"len": func(s interface{}) int {
				switch v := s.(type) {
				case string:
					return len(v)
				case []domain.ConversationTurn:
					return len(v)
				case []*domain.Issue:
					return len(v)
				case []domain.RunningEntry:
					return len(v)
				case []domain.RetryEntry:
					return len(v)
				default:
					return 0
				}
			},
		}).
		ParseFS(templateFS, "layout.html", "partials/*.html", "pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &TemplateEngine{
		templates: tmpl,
		fs:        templateFS,
	}, nil
}

// Render 渲染模板到字符串
func (e *TemplateEngine) Render(name string, data *TemplateData) (string, error) {
	var buf strings.Builder
	err := e.templates.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.String(), nil
}

// RenderHTML 使用 Gin 渲染 HTML 响应
func (e *TemplateEngine) RenderHTML(c *gin.Context, name string, data *TemplateData) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	html, err := e.Render(name, data)
	if err != nil {
		c.String(http.StatusInternalServerError, "模板渲染错误: "+err.Error())
		return
	}
	c.String(http.StatusOK, html)
}

// RenderPartial 渲染片段模板
func (e *TemplateEngine) RenderPartial(w io.Writer, name string, data *TemplateData) error {
	return e.templates.ExecuteTemplate(w, name, data)
}

// GlobalTemplateEngine 全局模板引擎实例（可选）
var GlobalTemplateEngine *TemplateEngine

// InitTemplateEngine 初始化全局模板引擎
func InitTemplateEngine(fs embed.FS) error {
	engine, err := NewTemplateEngine(fs)
	if err != nil {
		return err
	}
	GlobalTemplateEngine = engine
	return nil
}

// GetTemplateEngine 获取全局模板引擎
func GetTemplateEngine() *TemplateEngine {
	return GlobalTemplateEngine
}