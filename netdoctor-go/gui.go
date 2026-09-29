// NetDoctor GUI（walk / 原生 Win32，Win7+ 兼容，零运行时依赖）
// 布局：顶部工具栏 | 检测结果列表/详情 | 底部状态栏
// 检测项逐项实时出现，5 种状态色；报告可导出 HTML / JSON / TXT
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// ---------- 检测项表格模型 ----------
type itemModel struct {
	walk.TableModelBase
	items []CheckItem
}

func (m *itemModel) RowCount() int { return len(m.items) }

func (m *itemModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.items) {
		return ""
	}
	it := m.items[row]
	switch col {
	case 0:
		return statusMark(it.Status) + " " + statusText(it.Status)
	case 1:
		return it.Name
	case 2:
		return it.Value
	}
	return ""
}

// itemStyler 状态列着色（PASS 绿 / WARN 橙 / FAIL 红 / RUNNING 蓝 / PENDING 灰）
type itemStyler struct{ m *itemModel }

func (s *itemStyler) StyleCell(style *walk.CellStyle) {
	if style.Col() != 0 {
		return
	}
	row := style.Row()
	if row < 0 || row >= len(s.m.items) {
		return
	}
	switch s.m.items[row].Status {
	case StatusPass:
		style.TextColor = walk.RGB(0x16, 0xa3, 0x4a)
	case StatusWarn:
		style.TextColor = walk.RGB(0xd9, 0x77, 0x06)
	case StatusFail:
		style.TextColor = walk.RGB(0xdc, 0x26, 0x26)
	case StatusRunning:
		style.TextColor = walk.RGB(0x1d, 0x4e, 0xd8)
	default:
		style.TextColor = walk.RGB(0x64, 0x74, 0x8b)
	}
}

// ---------- 主窗口控制器 ----------
type appUI struct {
	mw         *walk.MainWindow
	model      *itemModel
	table      *walk.TableView
	detail     *walk.TextEdit
	statusLbl  *walk.Label
	progress   *walk.ProgressBar
	elapsed    *walk.Label
	exportBtn  *walk.PushButton
	redetect   *walk.PushButton
	formatBox  *walk.ComboBox
	pingAddr   *walk.LineEdit
	pingResult *walk.TextEdit
	pingBtn    *walk.PushButton
	pinging    bool
	running    bool
	lastRes    *CheckResult
	startAt    time.Time
}

// sync 切到 UI 线程（窗口销毁后安全忽略）
func (a *appUI) sync(fn func()) {
	if a.mw == nil {
		return
	}
	a.mw.Synchronize(func() {
		defer func() { _ = recover() }()
		fn()
	})
}

// startDetect 启动检测 goroutine
func (a *appUI) startDetect() {
	if a.running {
		return
	}
	a.running = true
	a.startAt = time.Now()
	a.sync(func() {
		a.redetect.SetEnabled(false)
		a.exportBtn.SetEnabled(false)
		a.model.items = nil
		a.model.PublishRowsReset()
		a.detail.SetText("")
		a.progress.SetValue(0)
		a.statusLbl.SetText("准备检测...")
	})

	// 耗时显示
	go func() {
		for a.running {
			time.Sleep(500 * time.Millisecond)
			a.sync(func() {
				a.elapsed.SetText("耗时 " + time.Since(a.startAt).Round(time.Second).String())
			})
		}
	}()

	go func() {
		defer func() { a.running = false }()
		res := detectAll(
			func(stage string, pct int) {
				a.sync(func() {
					a.statusLbl.SetText(stage)
					a.progress.SetValue(pct)
				})
			},
			func(item CheckItem, pct int) {
				a.sync(func() {
					// 逐项实时出现
					a.model.items = append(a.model.items, item)
					a.model.PublishRowsReset()
					a.progress.SetValue(pct)
					a.statusLbl.SetText(item.Name)
				})
			},
		)
		a.sync(func() {
			a.lastRes = res
			a.exportBtn.SetEnabled(true)
			a.redetect.SetEnabled(true)
			a.progress.SetValue(100)
			a.statusLbl.SetText("检测完成，共 " + fmt.Sprint(len(res.Items)) + " 项")
			// 检测完成弹窗：是否导出（回车=是）
			formats := []string{"HTML 报告", "JSON 数据", "TXT 文本"}
			fi := a.formatBox.CurrentIndex()
			if fi < 0 || fi > 2 {
				fi = 0
			}
			r := walk.MsgBox(a.mw, "检测完成",
				crlf("检测完成，共 "+fmt.Sprint(len(res.Items))+" 项。\n\n是否立即导出检测报告？\n当前格式："+formats[fi]),
				walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
			if r == 6 { // IDYES
				a.exportToDesktop()
			}
		})
	}()
}

// showDetail 选中检测项 → 详情区
func (a *appUI) showDetail() {
	idx := a.table.CurrentIndex()
	if idx < 0 || idx >= len(a.model.items) {
		return
	}
	d := a.model.items[idx].Detail
	if d == "" {
		d = "（无补充说明）"
	}
	a.detail.SetText(crlf(d))
}

// doPing 右侧 Ping 面板：输入 IP/域名，后台执行，人话结果回填（不卡界面）
func (a *appUI) doPing() {
	if a.pinging {
		return
	}
	addr := a.pingAddr.Text()
	if addr == "" {
		a.pingResult.SetText("请输入要检测的 IP 地址或域名。")
		a.pingAddr.SetFocus()
		return
	}
	a.pinging = true
	a.pingBtn.SetEnabled(false)
	a.pingBtn.SetText("检测中…")
	a.pingResult.SetTextColor(walk.RGB(0x0f, 0x17, 0x2a))
	a.pingResult.SetText("正在检测 " + addr + " ……")
	go func() {
		r := pingStats(addr, 4, 1000)
		text := crlf(r.pingSummary())
		color, mark := pingColorClass(&r)
		a.sync(func() {
			a.pingResult.SetTextColor(color)
			a.pingResult.SetText(mark + " " + text)
			a.pinging = false
			a.pingBtn.SetEnabled(true)
			a.pingBtn.SetText("开始 Ping")
			a.pingAddr.SetFocus()
		})
	}()
}

// ---------- 导出 ----------
// writeReportByFormat 按当前所选格式写出报告
func (a *appUI) writeReportByFormat(path string) error {
	fi := a.formatBox.CurrentIndex()
	if fi < 0 || fi > 2 {
		fi = 0
	}
	switch fi {
	case 1:
		return os.WriteFile(path, []byte(buildJSONReport(a.lastRes)), 0644)
	case 2:
		return writeReportGBK(path, a.lastRes.Text)
	default:
		return os.WriteFile(path, []byte(buildHTMLReport(a.lastRes)), 0644)
	}
}

// crlf Windows 的 EDIT/MessageBox 控件不识别裸 \n，统一转为 \r\n 才能正确换行
func crlf(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

// pingColorClass 按 Ping 结果返回状态色与图标（PASS 绿 / WARN 橙 / FAIL 红）
func pingColorClass(r *PingResult) (walk.Color, string) {
	if !r.ResolveOK || r.Received == 0 {
		return walk.RGB(0xdc, 0x26, 0x26), "✗" // FAIL：解析失败或全部超时
	}
	if r.lossPercent() > 0 || r.AvgMs > 100 {
		return walk.RGB(0xd9, 0x77, 0x06), "⚠" // WARN：丢包或延迟偏高/高
	}
	return walk.RGB(0x16, 0xa3, 0x4a), "✓" // PASS：全通且延迟正常
}

// desktopDir 获取真实桌面目录（兼容桌面重定向）
func desktopDir() string {
	var buf [win.MAX_PATH]uint16
	if win.SHGetSpecialFolderPath(0, &buf[0], win.CSIDL_DESKTOPDIRECTORY, false) {
		return win.UTF16PtrToString(&buf[0])
	}
	return os.Getenv("USERPROFILE") + "\\Desktop"
}

// exportToDesktop 直接导出到桌面（检测完成弹窗"是"后调用）
func (a *appUI) exportToDesktop() {
	if a.lastRes == nil {
		return
	}
	exts := []string{"html", "json", "txt"}
	fi := a.formatBox.CurrentIndex()
	if fi < 0 || fi > 2 {
		fi = 0
	}
	path := desktopDir() + "\\网络环境检测报告_" + time.Now().Format("20060102150405") + "." + exts[fi]
	if err := a.writeReportByFormat(path); err != nil {
		walk.MsgBox(a.mw, "导出报告", "写入失败："+err.Error(), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(a.mw, "导出报告", crlf("报告已导出到：\n"+path), walk.MsgBoxIconInformation)
}

// exportReport 按所选格式导出（保存对话框选择路径）
func (a *appUI) exportReport() {
	if a.lastRes == nil {
		walk.MsgBox(a.mw, "导出报告", "尚无检测结果，请先完成检测。", walk.MsgBoxIconInformation)
		return
	}
	formats := []string{"HTML 报告 (*.html)", "JSON 数据 (*.json)", "TXT 文本 (*.txt)"}
	exts := []string{"html", "json", "txt"}
	fi := a.formatBox.CurrentIndex()
	if fi < 0 || fi >= len(formats) {
		fi = 0
	}
	dlg := new(walk.FileDialog)
	dlg.Title = "导出检测报告"
	dlg.FilePath = "网络环境检测报告_" + time.Now().Format("20060102150405") + "." + exts[fi]
	dlg.Filter = formats[fi] + "|*." + exts[fi] + "|所有文件 (*.*)|*.*"
	ok, err := dlg.ShowSave(a.mw)
	if err != nil {
		walk.MsgBox(a.mw, "导出报告", "保存对话框出错："+err.Error(), walk.MsgBoxIconError)
		return
	}
	if !ok {
		return
	}
	if werr := a.writeReportByFormat(dlg.FilePath); werr != nil {
		walk.MsgBox(a.mw, "导出报告", "写入失败："+werr.Error(), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(a.mw, "导出报告", crlf("报告已导出到：\n"+dlg.FilePath), walk.MsgBoxIconInformation)
}

// loadAppIcon 从 exe 自身提取嵌入的图标（.ico 资源），用于窗口标题栏与任务栏
func loadAppIcon() *walk.Icon {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	if ic, err := walk.NewIconExtractedFromFile(exe, 0, 0); err == nil {
		return ic
	}
	return nil
}

// ---------- 验证码对话框 ----------
func showTotpDialog() bool {
	var dlg *walk.Dialog
	var verifyBtn, cancelBtn *walk.PushButton
	var codeEdit *walk.LineEdit
	passed := false
	attempts := 0

	// doVerify 校验动态码（按钮点击与回车共用）
	doVerify := func() {
		if verifyTOTP(totpSecret(), codeEdit.Text()) {
			passed = true
			dlg.Accept()
			return
		}
		attempts++
		if attempts >= totpMaxAttempts {
			walk.MsgBox(dlg, "验证失败", "验证码错误次数过多，程序退出。", walk.MsgBoxIconError)
			dlg.Cancel()
			return
		}
		walk.MsgBox(dlg, "验证失败", "验证码错误或已过期，请重试。", walk.MsgBoxIconWarning)
		codeEdit.SetText("")
		codeEdit.SetFocus()
	}

	dialogCfg := Dialog{
		AssignTo:      &dlg,
		Title:         "NetDoctor - 权限验证",
		Icon:          loadAppIcon(),
		MinSize:       Size{Width: 440, Height: 190},
		DefaultButton: &verifyBtn,
		CancelButton:  &cancelBtn,
		Layout: VBox{
			Margins: Margins{14, 14, 14, 14},
			Spacing: 10,
		},
		Children: []Widget{
			Label{Text: "请输入验证码"},
			LineEdit{
				AssignTo:  &codeEdit,
				MaxLength: 6,
				Font:      Font{Family: "Consolas", PointSize: 18},
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn { // 输入完回车直接验证
						doVerify()
					}
				},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					PushButton{
						AssignTo:  &verifyBtn,
						Text:      "验证",
						OnClicked: doVerify,
					},
					PushButton{
						AssignTo:  &cancelBtn,
						Text:      "取消",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}

	if _, err := dialogCfg.Run(nil); err != nil {
		return false
	}
	return passed
}

// ---------- 主窗口 ----------
func runMainWindow() {
	app := &appUI{model: &itemModel{}}

	MainWindow{
		AssignTo: &app.mw,
		Title:    "NetDoctor 网络环境检测工具",
		Icon:     loadAppIcon(),
		Size:     Size{Width: 1024, Height: 800},
		MinSize:  Size{Width: 900, Height: 640},
		Layout: VBox{
			Margins: Margins{10, 10, 10, 10},
			Spacing: 8,
		},
		Children: []Widget{
			// 顶部工具栏
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{Text: "NetDoctor 网络环境检测工具", Font: Font{PointSize: 11}, MinSize: Size{Width: 220}},
					Label{Text: "导出格式:"},
					ComboBox{
						AssignTo:     &app.formatBox,
						Model:        []string{"HTML 报告", "JSON 数据", "TXT 文本"},
						CurrentIndex: 0,
					},
					PushButton{
						AssignTo:  &app.exportBtn,
						Text:      "导出报告",
						OnClicked: func() { app.exportReport() },
					},
					PushButton{
						AssignTo:  &app.redetect,
						Text:      "重新检测",
						OnClicked: func() { app.startDetect() },
					},
				},
			},
			// 主体：左 = 检测结果列表（上 8）+ 详情与建议（下 2）；右 = Ping 测试面板
			Composite{
				Layout:        HBox{Spacing: 8},
				StretchFactor: 1,
				Children: []Widget{
					Composite{
						Layout:        VBox{Spacing: 8},
						StretchFactor: 1,
						Children: []Widget{
							GroupBox{
								Title:         "检测结果",
								Layout:        VBox{},
								StretchFactor: 8,
								Children: []Widget{
									TableView{
										AssignTo: &app.table,
										Columns: []TableViewColumn{
											{Title: "状态", Width: 90},
											{Title: "检测项", Width: 200},
											{Title: "结果", Width: 420},
										},
										Model:                 app.model,
										OnCurrentIndexChanged: func() { app.showDetail() },
										StretchFactor:         1,
									},
								},
							},
							GroupBox{
								Title:         "详情与建议",
								Layout:        VBox{},
								StretchFactor: 2,
								MinSize:       Size{Height: 80},
								Children: []Widget{
									TextEdit{
										AssignTo:      &app.detail,
										ReadOnly:      true,
										VScroll:       true,
										Font:          Font{Family: "Consolas", PointSize: 9},
										StretchFactor: 1,
									},
								},
							},
						},
					},
					GroupBox{
						Title:   "Ping",
						Layout:  VBox{Spacing: 6},
						MinSize: Size{Width: 280},
						MaxSize: Size{Width: 300, Height: 380},
						Children: []Widget{
							Label{Text: "目标地址（IP 或域名）："},
							LineEdit{
								AssignTo: &app.pingAddr,
								Font:     Font{Family: "Consolas", PointSize: 12},
								OnKeyDown: func(key walk.Key) {
									if key == walk.KeyReturn {
										app.doPing()
									}
								},
							},
							PushButton{
								AssignTo:  &app.pingBtn,
								Text:      "开始 Ping",
								OnClicked: func() { app.doPing() },
							},
							Label{Text: "检测结果："},
							TextEdit{
								AssignTo: &app.pingResult,
								ReadOnly: true,
								VScroll:  true,
								Font:     Font{Family: "Microsoft YaHei", PointSize: 10},
								MinSize:  Size{Height: 160},
							},
							Label{Text: "提示：回车开始检测；共发送 4 个数据包。"},
						},
					},
				},
			},
			// 底部状态栏
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{AssignTo: &app.statusLbl, Text: "就绪", MinSize: Size{Width: 320}},
					ProgressBar{AssignTo: &app.progress, MinValue: 0, MaxValue: 100, StretchFactor: 1},
					Label{AssignTo: &app.elapsed, Text: "", MinSize: Size{Width: 110}},
				},
			},
		},
	}.Create()

	// 状态列着色
	app.table.SetCellStyler(&itemStyler{m: app.model})

	go app.startDetect()

	app.mw.Run()
}
