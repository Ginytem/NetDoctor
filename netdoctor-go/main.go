// NetDoctor - Windows 网络环境检测工具（Go GUI 版）
// 原版：Ginytem/NetDoctor 纯 BAT+VBScript（pack/net_diag.bat）
// 本版：Go 1.20 单文件 EXE，原生 Win32 GUI（walk），Win7 SP1 ~ Win11
// 验证码：本地构建（含 totp_local.go 密钥）启用 TOTP 动态验证码；公开版无密钥文件自动跳过。
package main

func main() {
	// 密钥已配置（本地版）→ 必须通过动态验证码才允许使用；公开版（无密钥）直接进入
	if totpEnabled() && !showTotpDialog() {
		return
	}

	runMainWindow()
}
