// 检测项模型：GUI 逐项实时展示 + 状态色 + 分类导航
package main

// 检测项状态（5 态）
const (
	StatusPending = iota // 待检测（灰）
	StatusRunning        // 检测中（蓝）
	StatusPass           // 通过（绿）
	StatusWarn           // 注意（橙）
	StatusFail           // 失败（红）
)

// statusText 状态显示文本
func statusText(s int) string {
	switch s {
	case StatusPending:
		return "待检测"
	case StatusRunning:
		return "检测中"
	case StatusPass:
		return "通过"
	case StatusWarn:
		return "注意"
	case StatusFail:
		return "失败"
	}
	return "未知"
}

// statusMark 状态符号（列表前缀）
func statusMark(s int) string {
	switch s {
	case StatusPending:
		return "○"
	case StatusRunning:
		return "●"
	case StatusPass:
		return "✓"
	case StatusWarn:
		return "▲"
	case StatusFail:
		return "✗"
	}
	return "?"
}

// 检测项分类（左侧导航）
const (
	CatHardware = "硬件配置"
	CatRuntime  = "运行库依赖"
	CatNetwork  = "网络结构"
	CatIPAssign = "IP 分配"
	CatExternal = "外网连通"
	CatLAN      = "局域网"
)

// CheckItem 单个检测项
type CheckItem struct {
	Category string // 所属分类（报告分节用）
	Name     string // 检测项名称
	Value    string // 检测值
	Status   int    // 状态
	Detail   string // 诊断细节（仅软件内详情区展示，不含对外导出）
	Advice   string // 对外建议文案（HTML 报告"说明"列用，不暴露检测机制）
}

// CheckResult 一次完整检测的全部结果
type CheckResult struct {
	Items   []CheckItem // 全部检测项
	Devices []LanDevice // 局域网设备（表格）
	NowStr  string
	Report  ReportData // 兼容字段（TXT/HTML 导出用）
	Text    string     // 旧格式 TXT 报告
}
