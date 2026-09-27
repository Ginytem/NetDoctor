// 阶段 4：报告生成
// 与原版 pack/net_diag.bat 的输出格式逐行对齐，文件保存为 GBK 编码
// （兼容 Win7 老版记事本直接打开不乱码）
package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

type ReportData struct {
	NowStr     string
	OSType     string
	OSArch     string
	OSName     string
	OSBuild    string
	CPU        string
	RAM        string
	Disk       string
	GPU        string
	VC64       string
	VC86       string
	DotNet     string
	WebView2   string
	Adapter    string
	NetType    string
	LocalIP    string
	Mask       string
	PrefixLen  int
	Gateway    string
	GWType     string
	DHCP       string
	DHCPServer string
	IPAdvice   string
	PubIP      string
	InetIP     string
	InetDNS    string
	LanDev     []string
	Cam        []string
	FreeIP     []string
}

func buildReport(d ReportData) string {
	var b strings.Builder
	b.WriteString("============================================================\n")
	b.WriteString("              网络环境检测报告\n")
	b.WriteString("============================================================\n")
	fmt.Fprintf(&b, "诊断时间: %s\n", d.NowStr)
	fmt.Fprintf(&b, "匹配策略: [%s 内核] [%s]\n", d.OSType, d.OSArch)
	b.WriteString("\n")
	b.WriteString("【客户端硬件配置】\n")
	fmt.Fprintf(&b, "  - 操作系统:  %s %s %s\n", d.OSName, d.OSArch, d.OSBuild)
	fmt.Fprintf(&b, "  - CPU:  %s\n", d.CPU)
	fmt.Fprintf(&b, "  - 运存:  %s\n", d.RAM)
	fmt.Fprintf(&b, "  - C 盘剩余:  %s\n", d.Disk)
	fmt.Fprintf(&b, "  - 显卡:  %s\n", d.GPU)
	b.WriteString("\n")
	b.WriteString("【必备运行库依赖】\n")
	fmt.Fprintf(&b, "  - VC++ 2015-2022（x64）: %s\n", d.VC64)
	fmt.Fprintf(&b, "  - VC++ 2015-2022（x86）: %s\n", d.VC86)
	fmt.Fprintf(&b, "  - .NET Framework 运行库: %s\n", d.DotNet)
	fmt.Fprintf(&b, "  - WebView2 Runtime: %s\n", d.WebView2)
	b.WriteString("\n")
	b.WriteString("------------------------------------------------------------\n")
	b.WriteString("【网络结构分析】\n")
	fmt.Fprintf(&b, "  - 活动网卡:  %s\n", d.Adapter)
	fmt.Fprintf(&b, "  - 上网方式:  %s\n", d.NetType)
	fmt.Fprintf(&b, "  - 本机 IP :  %s\n", d.LocalIP)
	fmt.Fprintf(&b, "  - 子网掩码:  %s（前缀 /%d）\n", d.Mask, d.PrefixLen)
	fmt.Fprintf(&b, "  - 默认网关:  %s\n", d.Gateway)
	fmt.Fprintf(&b, "  - 网关类型:  %s\n", d.GWType)
	b.WriteString("\n")
	b.WriteString("【IP 分配与建议】\n")
	fmt.Fprintf(&b, "  - 分配方式:  %s\n", d.DHCP)
	fmt.Fprintf(&b, "  - DHCP服务器:  %s\n", d.DHCPServer)
	fmt.Fprintf(&b, "  - 配置建议:  %s\n", d.IPAdvice)
	b.WriteString("\n")
	b.WriteString("【外网连通状态】\n")
	fmt.Fprintf(&b, "  - 出口 IPv4: %s\n", d.PubIP)
	fmt.Fprintf(&b, "  - IP 连通 : %s\n", d.InetIP)
	fmt.Fprintf(&b, "  - DNS 解析: %s\n", d.InetDNS)
	b.WriteString("\n")
	b.WriteString("【局域网发现设备】:\n")
	writeList(&b, d.LanDev)
	b.WriteString("\n")
	b.WriteString("【目标相机状态】:\n")
	writeList(&b, d.Cam)
	b.WriteString("\n")
	b.WriteString("【建议分配 IP】:\n")
	writeList(&b, d.FreeIP)
	b.WriteString("\n")
	b.WriteString("============================================================\n")
	b.WriteString("本报告仅呈现本次检测结果与建议操作，不包含诊断过程。检测结果基于当前网络状态，可能随环境变化。\n")
	b.WriteString("============================================================\n")
	return b.String()
}

func writeList(b *strings.Builder, lines []string) {
	if len(lines) == 0 {
		return
	}
	for _, l := range lines {
		b.WriteString(l)
		b.WriteString("\n")
	}
}

// 写 GBK 编码报告（CRLF 行尾，兼容老版记事本）
func writeReportGBK(path, text string) error {
	text = strings.ReplaceAll(text, "\n", "\r\n")
	enc := simplifiedchinese.GBK.NewEncoder()
	out, err := enc.String(text)
	if err != nil {
		out = text
	}
	return os.WriteFile(path, []byte(out), 0644)
}
