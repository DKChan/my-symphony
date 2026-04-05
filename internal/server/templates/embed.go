package templates

import "embed"

// TemplateFS 嵌入所有模板文件
//
//go:embed layout.html
//go:embed partials/*.html
//go:embed pages/*.html
var TemplateFS embed.FS