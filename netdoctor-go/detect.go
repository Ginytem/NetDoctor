// 检测流程编排：逐项产出检测项（CheckItem），实时上报给 GUI
package main

import (
	"fmt"
	"strings"
	"time"
)

// ItemFunc 单个检测项完成回调（在工作 goroutine 内调用；pct 为整体进度 0-100）
type ItemFunc func(item CheckItem, pct int)

// detectAll 执行完整检测，每完成一项立即回调 onItem
func detectAll(onProgress func(stage string, pct int), onItem ItemFunc) *CheckResult {
	now := time.Now()
	nowstr := now.Format("2006-01-02 15:04:05")
	res := &CheckResult{NowStr: nowstr}
	total := 22

	progress := func(stage string, pct int) {
		if onProgress != nil {
			onProgress(stage, pct)
		}
	}
	emit := func(item CheckItem, stage string, pct int) {
		res.Items = append(res.Items, item)
		progress(stage, pct)
		if onItem != nil {
			onItem(item, pct)
		}
	}
	// pct 计算：idx 从 1 开始
	var pctOf = func(idx int) int {
		return idx * 100 / total
	}

	// ===== 硬件配置 =====
	progress("正在检测硬件配置...", pctOf(0))
	si := collectSysInfo()

	emit(CheckItem{CatHardware, "操作系统", fmt.Sprintf("%s %s (Build %s)", si.OSName, si.OSArch, si.OSBuild), StatusPass,
		"操作系统版本：\n  " + si.OSType + " 内核\n  " + si.OSName + " " + si.OSArch + " (Build " + si.OSBuild + ")",
		"系统版本正常，无需操作。"}, "硬件配置检测中", pctOf(1))

	emit(CheckItem{CatHardware, "CPU", si.CPUName, StatusPass,
		"中央处理器：\n  " + si.CPUName,
		"CPU 识别正常。"}, "硬件配置检测中", pctOf(2))

	emit(CheckItem{CatHardware, "运存", si.RAMSize, StatusPass,
		"物理内存总量：\n  " + si.RAMSize,
		"内存容量正常，满足日常使用。"}, "硬件配置检测中", pctOf(3))

	diskSt := StatusPass
	if strings.Contains(si.DiskFree, "GB") {
		diskSt = StatusPass
	}
	emit(CheckItem{CatHardware, "C 盘剩余", si.DiskFree, diskSt,
		"系统盘剩余空间：\n  " + si.DiskFree,
		"建议系统盘至少保留 2GB 以上可用空间，避免影响系统运行。"}, "硬件配置检测中", pctOf(4))

	emit(CheckItem{CatHardware, "显卡", si.GPUName, StatusPass,
		"显示适配器：\n  " + si.GPUName,
		"显示设备识别正常。"}, "硬件配置检测中", pctOf(5))

	// ===== 运行库依赖 =====
	progress("正在检测运行库依赖...", pctOf(5))
	rt := collectRuntimes()

	runtimeItem := func(name, val, advice string) CheckItem {
		st := StatusPass
		if strings.Contains(val, "未安装") {
			st = StatusFail
		}
		return CheckItem{CatRuntime, name, val, st, "注册表查询结果：\n  " + val, advice}
	}
	emit(runtimeItem("VC++ 2015-2022（x64）", rt.VC64, "运行库缺失时部分软件可能无法启动，如遇此情况请安装 Microsoft Visual C++ 运行库。"), "运行库检测中", pctOf(6))
	emit(runtimeItem("VC++ 2015-2022（x86）", rt.VC86, "运行库缺失时部分软件可能无法启动，如遇此情况请安装 Microsoft Visual C++ 运行库。"), "运行库检测中", pctOf(7))
	emit(runtimeItem(".NET Framework", rt.DotNet, ".NET 环境缺失时依赖其运行的软件可能无法启动，建议安装对应版本。"), "运行库检测中", pctOf(8))
	emit(runtimeItem("WebView2 Runtime", rt.WebView2, "个别新版应用依赖此组件，缺失时建议安装以保障兼容。"), "运行库检测中", pctOf(9))

	// ===== 网络结构 =====
	progress("正在读取网卡、IP 与网关...", pctOf(9))
	net := collectNetwork()

	netItem := func(name, val string, st int, detail, advice string) CheckItem {
		return CheckItem{CatNetwork, name, val, st, detail, advice}
	}
	adapterSt := StatusPass
	if net.Adapter == "" {
		adapterSt = StatusFail
	}
	emit(netItem("活动网卡", net.Adapter, adapterSt,
		"当前上网使用的网卡：\n  "+net.Adapter,
		"当前通过「"+net.Adapter+"」上网。"), "网络结构检测中", pctOf(10))

	emit(netItem("上网方式", net.NetType, StatusPass,
		"上网连接方式：\n  "+net.NetType,
		"当前为"+net.NetType+"。"), "网络结构检测中", pctOf(11))

	ipSt := StatusPass
	if net.LocalIP == "" || net.LocalIP == "127.0.0.1" {
		ipSt = StatusFail
	}
	emit(netItem("本机 IP", net.LocalIP, ipSt,
		"本机 IPv4 地址：\n  "+net.LocalIP,
		"本机 IP 正常；若显示 127.0.0.1 表示未获取到有效地址，请检查网线连接与网卡状态。"), "网络结构检测中", pctOf(12))

	emit(netItem("子网掩码", fmt.Sprintf("%s（前缀 /%d）", net.Mask, net.PrefixLen), StatusPass,
		"子网掩码与前缀长度：\n  "+net.Mask+"（前缀 /"+fmt.Sprint(net.PrefixLen)+"）",
		"网段划分正常。"), "网络结构检测中", pctOf(13))

	gwSt := StatusPass
	if net.Gateway == "" {
		gwSt = StatusFail
	}
	emit(netItem("默认网关", net.Gateway, gwSt,
		"默认网关地址：\n  "+net.Gateway,
		"网关正常；若为空请检查路由器供电、网线连接与上级网络。"), "网络结构检测中", pctOf(14))

	// ===== 网关类型（较耗时：HTTP 指纹 + tracert）=====
	progress("正在判定网关类型...", pctOf(14))
	gwtype := determineGatewayType(net.Gateway)
	gwtSt := StatusPass
	if strings.Contains(gwtype, "未识别") || strings.Contains(gwtype, "疑似") {
		gwtSt = StatusWarn
	}
	emit(netItem("网关类型", gwtype, gwtSt,
		"网关设备综合判定结果：\n  "+gwtype,
		"网关类型： "+gwtype+"。"), "网关类型判定中", pctOf(15))

	// ===== IP 分配 =====
	progress("正在分析 IP 分配方式...", pctOf(15))
	dhcpSt := StatusPass
	if net.DHCP == "手动指定（静态）" {
		dhcpSt = StatusWarn
	}
	emit(CheckItem{CatIPAssign, "分配方式", net.DHCP, dhcpSt,
		"IP 获取方式：\n  " + net.DHCP,
		"IP 获取方式：" + net.DHCP + "。静态 IP 环境下请谨慎修改网络设置。"}, "IP 分配分析中", pctOf(16))

	emit(CheckItem{CatIPAssign, "DHCP 服务器", net.DHCPServer, StatusPass,
		"DHCP 服务器地址：\n  " + net.DHCPServer,
		"DHCP 服务正常。"}, "IP 分配分析中", pctOf(17))

	// ===== 外网连通 =====
	progress("正在检测外网连通性...", pctOf(17))
	inetIP := "不通"
	if pingOK("223.5.5.5", 1, 1200) {
		inetIP = "正常"
	}
	ipConnSt := StatusPass
	if inetIP == "不通" {
		ipConnSt = StatusFail
	}
	emit(CheckItem{CatExternal, "IP 连通", inetIP, ipConnSt,
		"公网连通性测试结果：\n  " + inetIP,
		"外网连通正常；若不通请检查宽带账号拨号状态、光猫指示灯与网线。"}, "外网连通检测中", pctOf(18))

	inetDNS := "异常"
	if pingOK("www.baidu.com", 1, 1500) {
		inetDNS = "正常"
	}
	dnsSt := StatusPass
	if inetDNS == "异常" {
		dnsSt = StatusFail
	}
	emit(CheckItem{CatExternal, "DNS 解析", inetDNS, dnsSt,
		"域名解析与外网访问测试结果：\n  " + inetDNS,
		"域名解析正常；若异常可尝试在路由器中更换 DNS 或重启光猫/路由器。"}, "外网连通检测中", pctOf(19))

	pub := publicIP()
	pubSt := StatusPass
	if pub == "" {
		pub = "无法获取"
		pubSt = StatusWarn
	}
	emit(CheckItem{CatExternal, "出口 IPv4", pub, pubSt,
		"本机公网出口 IP：\n  " + pub,
		"公网出口 IP：" + pub + "。"}, "外网连通检测中", pctOf(20))

	// ===== 局域网 =====
	progress("正在扫描局域网设备...", pctOf(20))
	gwseg := segOf(net.Gateway)
	if gwseg == "" {
		gwseg = segOf(net.LocalIP)
	}
	devs := lanDevicesStructured(net.LocalIP, net.Gateway, gwseg)
	lanSt := StatusPass
	if len(devs) == 0 {
		lanSt = StatusWarn
	}
	emit(CheckItem{CatLAN, "局域网设备", fmt.Sprintf("发现 %d 台", len(devs)), lanSt,
		"发现同网段在线设备 " + fmt.Sprintf("%d 台（详见设备表）。", len(devs)),
		"当前网段发现 " + fmt.Sprintf("%d 台在线设备，MAC 与厂商信息详见下方设备表。", len(devs)) + "若预期设备未出现在列表中，请检查该设备供电、网线连接与交换机端口状态。"}, "局域网扫描中", pctOf(21))

	progress("正在探测目标相机与空闲 IP...", pctOf(21))
	cam := cameraStatus(gwseg, net.LocalIP)
	camOn := 0
	for _, l := range cam {
		if strings.Contains(l, "在线") {
			camOn++
		}
	}
	camSt := StatusWarn
	camSummary := fmt.Sprintf("%d 台无响应", 4-camOn)
	if camOn == 4 {
		camSt = StatusPass
		camSummary = "全部在线"
	}
	emit(CheckItem{CatLAN, "目标相机状态", camSummary, camSt,
		"相机探测明细：\n" + strings.Join(cam, "\n"),
		"已探测 4 台目标相机（.100/.101/.108/.64），各相机 ICMP 与二层应答状态详见下方相机明细表；多数工业/监控相机默认禁 ping，若相机实际在线但显示无响应，请用厂商 IP 搜索工具或 ONVIF 探测确认。"}, "相机探测中", pctOf(22))

	baseIP := net.Gateway
	if baseIP == "" {
		baseIP = net.LocalIP
	}
	freeIP := freeIPs(baseIP, net.LocalIP, net.Gateway, net.PrefixLen)
	freeSt := StatusPass
	if len(freeIP) == 1 && strings.Contains(freeIP[0], "未扫描") {
		freeSt = StatusWarn
	}
	freeText := strings.Join(freeIP, "\n")
	emit(CheckItem{CatLAN, "建议分配 IP", fmt.Sprintf("%d 个空闲地址", len(freeIP)), freeSt,
		"可用的空闲地址：\n" + freeText,
		"可选用以下空闲地址为新增设备分配 IP：\n" + freeText}, "空闲 IP 扫描中", pctOf(22))

	// ===== 汇总 ReportData（兼容 TXT/HTML 导出）=====
	ipAdvice := "局域网采用 DHCP 动态分配，可直接选用后段空闲 IP"
	if net.DHCP == "手动指定（静态）" {
		ipAdvice = "局域网可能存在严格固定 IP 规则，建议向现场管理员报备"
	}
	res.Report = ReportData{
		NowStr:     nowstr,
		OSType:     si.OSType,
		OSArch:     si.OSArch,
		OSName:     si.OSName,
		OSBuild:    si.OSBuild,
		CPU:        si.CPUName,
		RAM:        si.RAMSize,
		Disk:       si.DiskFree,
		GPU:        si.GPUName,
		VC64:       rt.VC64,
		VC86:       rt.VC86,
		DotNet:     rt.DotNet,
		WebView2:   rt.WebView2,
		Adapter:    net.Adapter,
		NetType:    net.NetType,
		LocalIP:    net.LocalIP,
		Mask:       net.Mask,
		PrefixLen:  net.PrefixLen,
		Gateway:    net.Gateway,
		GWType:     gwtype,
		DHCP:       net.DHCP,
		DHCPServer: net.DHCPServer,
		IPAdvice:   ipAdvice,
		PubIP:      pub,
		InetIP:     inetIP,
		InetDNS:    inetDNS,
		LanDev:     formatLanDevices(devs),
		Cam:        cam,
		FreeIP:     freeIP,
	}
	res.Devices = devs
	res.Text = buildReport(res.Report)
	progress("检测完成", 100)
	return res
}
