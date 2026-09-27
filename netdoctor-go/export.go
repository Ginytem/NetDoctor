// 报告导出：HTML（正式文档样式，双击可开）+ JSON（结构化数据）
// 另保留 GBK TXT（原版格式，兼容老记事本）
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// statusCSS 状态 -> 颜色（与界面统一）
func statusCSS(s int) string {
	switch s {
	case StatusPass:
		return "#22c55e"
	case StatusWarn:
		return "#f59e0b"
	case StatusFail:
		return "#ef4444"
	case StatusRunning:
		return "#3b82f6"
	default:
		return "#94a3b8"
	}
}

// statusJSON 状态 -> 文本（JSON 用）
func statusJSON(s int) string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusWarn:
		return "WARN"
	case StatusFail:
		return "FAIL"
	case StatusRunning:
		return "RUNNING"
	default:
		return "PENDING"
	}
}

// ---------- HTML ----------

// buildHTMLReport 生成独立 HTML 报告（内联样式，可打印）
func buildHTMLReport(res *CheckResult) string {
	pass, warn, fail := 0, 0, 0
	for _, it := range res.Items {
		switch it.Status {
		case StatusPass:
			pass++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		}
	}

	// 分类汇总（按固定分类顺序）
	catOrder := []string{CatHardware, CatRuntime, CatNetwork, CatIPAssign, CatExternal, CatLAN}
	catCount := map[string][3]int{}
	for _, it := range res.Items {
		c := catCount[it.Category]
		switch it.Status {
		case StatusPass:
			c[0]++
		case StatusWarn:
			c[1]++
		case StatusFail:
			c[2]++
		}
		catCount[it.Category] = c
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>网络环境检测报告</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: "Microsoft YaHei", "Segoe UI", sans-serif; background: #f8fafc; color: #0f172a; padding: 32px; }
  .report { max-width: 960px; margin: 0 auto; background: #fff; border-radius: 6px; padding: 40px 48px; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
  h1 { font-size: 24px; margin-bottom: 4px; }
  .meta { color: #64748b; font-size: 13px; margin-bottom: 24px; }
  .summary { display: flex; gap: 16px; margin-bottom: 28px; }
  .stat { flex: 1; border-radius: 6px; padding: 16px 20px; background: #f1f5f9; }
  .stat .num { font-size: 32px; font-weight: 700; }
  .stat .label { font-size: 13px; color: #64748b; margin-top: 4px; }
  .stat.pass .num { color: #22c55e; }
  .stat.warn .num { color: #f59e0b; }
  .stat.fail .num { color: #ef4444; }
  h2 { font-size: 17px; margin: 26px 0 12px; padding-left: 10px; border-left: 4px solid #2563eb; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 8px; }
  th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #e2e8f0; font-size: 14px; vertical-align: top; }
  th { background: #f8fafc; color: #64748b; font-weight: 600; white-space: nowrap; }
  td.name { width: 30%; font-weight: 600; }
  td.status { white-space: nowrap; font-weight: 600; }
  .tag { display: inline-block; padding: 1px 8px; border-radius: 10px; font-size: 12px; color: #fff; }
  .detail { font-size: 12.5px; color: #64748b; }
  .detail pre { font-family: Consolas, "JetBrains Mono", monospace; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 4px; padding: 10px; white-space: pre-wrap; word-break: break-all; margin-top: 6px; }
  .note { font-size: 12.5px; color: #64748b; margin: 6px 0; }
  .footer { margin-top: 28px; padding-top: 16px; border-top: 1px solid #e2e8f0; font-size: 12px; color: #94a3b8; }
  @media print { body { padding: 0; background: #fff; } .report { box-shadow: none; padding: 20px; } .stat { break-inside: avoid; } }
</style>
</head>
<body>
<div class="report">
<h1>网络环境检测报告</h1>
<div class="meta">生成时间：` + res.NowStr + `</div>
<div class="summary">
  <div class="stat pass"><div class="num">` + fmt.Sprint(pass) + `</div><div class="label">通过 PASS</div></div>
  <div class="stat warn"><div class="num">` + fmt.Sprint(warn) + `</div><div class="label">注意 WARN</div></div>
  <div class="stat fail"><div class="num">` + fmt.Sprint(fail) + `</div><div class="label">失败 FAIL</div></div>
</div>
`)

	for _, cat := range catOrder {
		var items []CheckItem
		for _, it := range res.Items {
			if it.Category == cat {
				items = append(items, it)
			}
		}
		if len(items) == 0 {
			continue
		}
		c := catCount[cat]
		b.WriteString("<h2>" + html.EscapeString(cat) + ` <span style="font-size:12px;color:#64748b;font-weight:400">（` +
			fmt.Sprintf("%d 通过 / %d 注意 / %d 失败", c[0], c[1], c[2]) + `）</span></h2>`)
		b.WriteString(`<table><tr><th>检测项</th><th>结果</th><th>状态</th><th>说明</th></tr>`)
		for _, it := range items {
			b.WriteString("<tr><td class=\"name\">" + html.EscapeString(it.Name) + "</td>")
			b.WriteString("<td>" + html.EscapeString(it.Value) + "</td>")
			b.WriteString("<td class=\"status\"><span class=\"tag\" style=\"background:" + statusCSS(it.Status) + "\">" + statusText(it.Status) + "</span></td>")
			b.WriteString("<td class=\"detail\">" + htmlDetail(it.Advice) + "</td></tr>")
		}
		b.WriteString("</table>")
	}

	// 局域网设备表
	if len(res.Devices) > 0 {
		b.WriteString(`<h2>局域网发现设备</h2><table><tr><th>IP</th><th>MAC</th><th>厂商</th><th>状态</th><th>来源</th></tr>`)
		for _, d := range res.Devices {
			b.WriteString("<tr><td>" + html.EscapeString(d.IP) + "</td><td>" + html.EscapeString(d.MAC) + "</td><td>" + html.EscapeString(d.Vendor) + "</td><td>" + html.EscapeString(d.Status) + "</td><td>" + html.EscapeString(d.Source) + "</td></tr>")
		}
		b.WriteString("</table>")
	}

	// 目标相机状态表（逐台列出探测结果）
	if len(res.Report.Cam) > 0 {
		b.WriteString(`<h2>目标相机状态</h2>`)
		hasRow := false
		for _, l := range res.Report.Cam {
			if ip, _ := parseCamLine(l); ip != "" {
				hasRow = true
				break
			}
		}
		if hasRow {
			b.WriteString(`<table><tr><th>相机 IP</th><th>状态</th></tr>`)
			for _, l := range res.Report.Cam {
				ip, st := parseCamLine(l)
				if ip == "" {
					continue
				}
				cls := "warn"
				if strings.Contains(st, "在线") {
					cls = "pass"
				}
				b.WriteString("<tr><td>" + html.EscapeString(ip) + "</td><td><span class=\"tag\" style=\"background:" +
					map[string]string{"pass": "#22c55e", "warn": "#f59e0b"}[cls] + "\">" + html.EscapeString(st) + "</span></td></tr>")
			}
			b.WriteString("</table>")
		}
		for _, l := range res.Report.Cam { // 注释说明行（如"注：多数工业/监控相机..."）
			if ip, _ := parseCamLine(l); ip == "" {
				b.WriteString("<p class=\"note\">" + html.EscapeString(strings.TrimSpace(l)) + "</p>")
			}
		}
	}

	// 建议分配 IP 列表
	if len(res.Report.FreeIP) > 0 {
		b.WriteString(`<h2>建议分配 IP</h2>`)
		var ips []string
		for _, l := range res.Report.FreeIP {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "- ") {
				ips = append(ips, strings.TrimSpace(strings.TrimPrefix(t, "- ")))
			} else if t != "" {
				b.WriteString("<p class=\"note\">" + html.EscapeString(t) + "</p>")
			}
		}
		if len(ips) > 0 {
			b.WriteString(`<table><tr><th>序号</th><th>空闲地址</th></tr>`)
			for i, ip := range ips {
				b.WriteString("<tr><td>" + fmt.Sprint(i+1) + "</td><td>" + html.EscapeString(ip) + "</td></tr>")
			}
			b.WriteString("</table>")
		}
	}

	b.WriteString(`<div class="footer">本报告仅呈现本次检测结果与建议操作，不包含诊断过程。检测结果基于当前网络状态，可能随环境变化。<br>NetDoctor 网络环境检测工具</div>
</div>
</body>
</html>`)
	return b.String()
}

// htmlDetail 详情文本转 HTML（\n → <br>，pre 保留原始输出风格）
func htmlDetail(d string) string {
	if d == "" {
		return "-"
	}
	var b strings.Builder
	lines := strings.Split(d, "\n")
	for i, ln := range lines {
		if i > 0 {
			b.WriteString("<br>")
		}
		b.WriteString(html.EscapeString(ln))
	}
	return b.String()
}

// parseCamLine 解析相机探测行 "    - 192.168.1.100  -- 在线（ping 通信正常）"
// 非 IP 行（注释/说明）返回 ip=""，state=""
func parseCamLine(l string) (ip, state string) {
	t := strings.TrimSpace(l)
	if !strings.HasPrefix(t, "- ") {
		return "", ""
	}
	t = strings.TrimPrefix(t, "- ")
	if i := strings.Index(t, "  -- "); i >= 0 {
		return strings.TrimSpace(t[:i]), strings.TrimSpace(t[i+len("  -- "):])
	}
	return t, ""
}

// ---------- JSON ----------

type jsonItem struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Advice string `json:"advice"`
}

type jsonCategory struct {
	Name  string     `json:"name"`
	Items []jsonItem `json:"items"`
}

type jsonDevice struct {
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Vendor string `json:"vendor"`
	Status string `json:"status"`
	Source string `json:"source"`
}

type jsonReport struct {
	Tool        string          `json:"tool"`
	Version     string          `json:"version"`
	GeneratedAt string          `json:"generated_at"`
	Summary     map[string]int  `json:"summary"`
	Categories  []jsonCategory `json:"categories"`
	LanDevices  []jsonDevice   `json:"lan_devices"`
}

// buildJSONReport 生成结构化 JSON 报告
func buildJSONReport(res *CheckResult) string {
	summary := map[string]int{"pass": 0, "warn": 0, "fail": 0}
	for _, it := range res.Items {
		switch it.Status {
		case StatusPass:
			summary["pass"]++
		case StatusWarn:
			summary["warn"]++
		case StatusFail:
			summary["fail"]++
		}
	}
	catOrder := []string{CatHardware, CatRuntime, CatNetwork, CatIPAssign, CatExternal, CatLAN}
	var cats []jsonCategory
	for _, cat := range catOrder {
		var items []jsonItem
		for _, it := range res.Items {
			if it.Category == cat {
				items = append(items, jsonItem{it.Name, it.Value, statusJSON(it.Status), it.Detail, it.Advice})
			}
		}
		if len(items) > 0 {
			cats = append(cats, jsonCategory{cat, items})
		}
	}
	devs := make([]jsonDevice, 0, len(res.Devices))
	for _, d := range res.Devices {
		devs = append(devs, jsonDevice{d.IP, d.MAC, d.Vendor, d.Status, d.Source})
	}
	jr := jsonReport{
		Tool:        "NetDoctor",
		Version:     "2.0-go-gui",
		GeneratedAt: res.NowStr,
		Summary:     summary,
		Categories:  cats,
		LanDevices:  devs,
	}
	out, err := json.MarshalIndent(jr, "", "  ")
	if err != nil {
		return `{"error":"` + err.Error() + `"}`
	}
	return string(out)
}
