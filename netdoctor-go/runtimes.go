// 阶段 1：必备运行库检测（VC++ / .NET Framework / WebView2）
package main

import (
	"golang.org/x/sys/windows/registry"
)

type Runtimes struct {
	VC64     string
	VC86     string
	DotNet   string
	WebView2 string
}

func collectRuntimes() Runtimes {
	rt := Runtimes{
		VC64:     "【缺失】",
		VC86:     "【缺失】",
		DotNet:   "【缺失/未安装 .NET 4.5+】",
		WebView2: "【缺失】",
	}
	if vcInstalled(`SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64`, keyWow6464Key) {
		rt.VC64 = "【已安装】"
	}
	if vcInstalled(`SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x86`, keyWow6432Key) {
		rt.VC86 = "【已安装】"
	}
	if v := dotNetVersion(); v != "" {
		rt.DotNet = v
	}
	if v := webview2Version(); v != "" {
		rt.WebView2 = v
	}
	return rt
}

// VC++ 2015-2022：Installed 值 = 1 即已安装（原版 reg query + findstr 0x1）
func vcInstalled(keyPath string, view uint32) bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE|view)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("Installed")
	return err == nil && v == 1
}

// .NET Framework 4.5~4.8.1 完整阈值表（原版逐级判断）
func dotNetVersion() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full`,
		registry.QUERY_VALUE|keyWow6464Key)
	if err != nil {
		return ""
	}
	defer k.Close()
	rel, _, err := k.GetIntegerValue("Release")
	if err != nil {
		return ""
	}
	switch {
	case rel >= 533320:
		return "【已安装】（.NET 4.8.1）"
	case rel >= 528040:
		return "【已安装】（.NET 4.8）"
	case rel >= 461808:
		return "【已安装】（.NET 4.7.2）"
	case rel >= 461308:
		return "【已安装】（.NET 4.7.1）"
	case rel >= 460798:
		return "【已安装】（.NET 4.7）"
	case rel >= 394802:
		return "【已安装】（.NET 4.6.2）"
	case rel >= 394271:
		return "【已安装】（.NET 4.6.1）"
	case rel >= 393295:
		return "【已安装】（.NET 4.6）"
	default:
		return "【已安装】（.NET 4.5.x）"
	}
}

// WebView2 Runtime：先查 32 位视图再查 64 位视图（原版 WOW6432Node 优先）
func webview2Version() string {
	guid := `SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	views := []uint32{keyWow6432Key, keyWow6464Key}
	for _, view := range views {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, guid, registry.QUERY_VALUE|view)
		if err != nil {
			continue
		}
		v, _, err := k.GetStringValue("pv")
		k.Close()
		if err == nil && v != "" {
			return "【已安装】(" + v + ")"
		}
	}
	return ""
}
