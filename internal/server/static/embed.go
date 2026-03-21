package static

import "embed"

// DashboardCSS 内嵌的 dashboard.css 文件
//
//go:embed dashboard.css
var DashboardFS embed.FS
