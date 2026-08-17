package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"syscall"
	"unsafe"
)

//go:embed payload.exe
var payload []byte

const (
	appTitle     = "XT-cursor用量小工具"
	setupTitle   = "XT-cursor用量小工具 安装程序"
	uninstTitle  = "XT-cursor用量小工具 卸载程序"
	appExeName   = "XT-cursor用量小工具.exe"
	uninstName   = "卸载.exe"
	appRegKey    = `Software\Microsoft\Windows\CurrentVersion\Uninstall\XTCursorUsage`
	publisher    = "XT"
	appVersion   = "1.2.4"

	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysmenu     = 0x00080000
	wsMinimizeBox = 0x00020000
	wsVisible     = 0x10000000
	wsChild       = 0x40000000
	wsTabstop     = 0x00010000
	wsClipChildren = 0x02000000
	esLeft        = 0
	esAutoHScroll = 0x0080
	wsExClientEdge = 0x00000200

	wmDestroy      = 0x0002
	wmClose        = 0x0010
	wmPaint        = 0x000F
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmSetCursor    = 0x0020
	wmCommand      = 0x0111
	wmEraseBk      = 0x0014
	wmCtlColorEdit = 0x0133
	wmSetFont      = 0x0030
	wmSetIcon      = 0x0080
	wmApp          = 0x8000
	wmAppStatus    = wmApp + 1
	wmAppDone      = wmApp + 2
	wmAppFail      = wmApp + 3
	wmAppStopped   = wmApp + 4

	swShow   = 5
	swHide   = 0
	swShowNA = 8

	idcArrow       = 32512
	idiApplication = 32512
	defaultCharset = 134
	clearTypeQual  = 5
	transparent    = 1
	opaque         = 2
	dtLeft         = 0
	dtCenter       = 1
	dtRight        = 2
	dtVCenter      = 4
	dtSingle       = 0x20
	dtWordBreak    = 0x10
	dtNoPrefix     = 0x800
	psSolid        = 0
	srcCopy        = 0x00CC0020
	mbOK           = 0
	mbIconError    = 0x10
	mbIconInfo     = 0x40
	smCxScreen     = 0
	smCyScreen     = 1
	swpNoZOrder    = 0x0004
	swpNoActivate  = 0x0010
	swpNoSize      = 0x0001
	swpNoMove      = 0x0002

	hkeyCU   = 0x80000001
	keyWrite = 0x20006
	regSZ    = 1
	regDWORD = 4

	bifReturnOnlyFSDirs = 0x00000001
	bifNewDialogStyle   = 0x00000040
	coInitApartment     = 0x2
	createNoWindow      = 0x08000000
	detachedProcess     = 0x00000008

	winW, winH int32 = 600, 520

	colBoard = 0x00381F05
	colCard  = 0x004F2E05
	colCyan  = 0x00E5D54A
	colWhite = 0x00F2F2F2
	colMute  = 0x0099A8B2
	colTrack = 0x002A3D4A
	colInk   = 0x00051F38
	colBtn   = 0x00624A18
	colInner = 0x00382408

	pageIntro    = 0
	pageFolder   = 1
	pageProgress = 2
	pageDone     = 3
	pageExist    = 4
	pageRunning  = 5
	pageUnAsk    = 0
	pageUnProg   = 1
	pageUnDone   = 2

	hitNext = 1
	hitBack = 2
	hitCancel = 3
	hitBrowse = 4
	hitDeskCB = 5
	hitFinish = 6
	hitUninst = 7
	hitUpdate = 8
	hitOverwrite = 9
	hitStop = 10
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procShowWindow         = user32.NewProc("ShowWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procLoadCursorW        = user32.NewProc("LoadCursorW")
	procSetCursor          = user32.NewProc("SetCursor")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procMessageBoxW        = user32.NewProc("MessageBoxW")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procBeginPaint         = user32.NewProc("BeginPaint")
	procEndPaint           = user32.NewProc("EndPaint")
	procFillRect           = user32.NewProc("FillRect")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procInvalidateRect     = user32.NewProc("InvalidateRect")
	procDrawTextW          = user32.NewProc("DrawTextW")
	procLoadIconW          = user32.NewProc("LoadIconW")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW  = user32.NewProc("GetWindowTextLengthW")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procUpdateWindow       = user32.NewProc("UpdateWindow")
	procFindWindowW        = user32.NewProc("FindWindowW")

	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procCreatePen             = gdi32.NewProc("CreatePen")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procCreateFontW           = gdi32.NewProc("CreateFontW")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procSetBkColor            = gdi32.NewProc("SetBkColor")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procRoundRect             = gdi32.NewProc("RoundRect")
	procRectangle             = gdi32.NewProc("Rectangle")
	procCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procBitBlt                = gdi32.NewProc("BitBlt")
	procDeleteDC              = gdi32.NewProc("DeleteDC")

	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	procGetModuleFileNameW = kernel32.NewProc("GetModuleFileNameW")
	procExtractIconExW     = shell32.NewProc("ExtractIconExW")
	procSHBrowseForFolderW = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	procRegCreateKeyExW    = advapi32.NewProc("RegCreateKeyExW")
	procRegOpenKeyExW      = advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW   = advapi32.NewProc("RegQueryValueExW")
	procRegSetValueExW     = advapi32.NewProc("RegSetValueExW")
	procRegCloseKey        = advapi32.NewProc("RegCloseKey")
	procRegDeleteKeyW      = advapi32.NewProc("RegDeleteKeyW")
	procCoInitializeEx     = ole32.NewProc("CoInitializeEx")
	procCoCreateInstance    = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree      = ole32.NewProc("CoTaskMemFree")
)

type wndClassEx struct {
	size, style            uint32
	wndProc                uintptr
	clsExtra, wndExtra     int32
	instance, icon, cursor syscall.Handle
	background             syscall.Handle
	menuName, className    *uint16
	iconSm                 syscall.Handle
}
type point struct{ x, y int32 }
type rect struct{ left, top, right, bottom int32 }
type msg struct {
	hwnd           syscall.Handle
	message        uint32
	wParam, lParam uintptr
	time           uint32
	pt             point
	private        uint32
}
type paintStruct struct {
	hdc                syscall.Handle
	erase              int32
	rcPaint            rect
	restore, incUpdate int32
	rgbReserved        [32]byte
}
type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}
type hit struct {
	id int
	r  rect
}

var (
	hInst, hwndMain, hwndEdit uintptr
	arrowCursor, appIcon      uintptr
	boardBrush, cardBrush     uintptr
	fontTitle, fontBody, fontSmall, fontBig uintptr
	hits                      []hit
	page                      int
	uninstallMode             bool
	busy                      bool
	wantDesktop               = true
	installDir                string
	hasExisting               bool
	existingDir, existingVer  string
	returnPage                int
	stopping                  bool
	progPct                   int
	progMsg                   string
	failMsg                   string
)

const introText = `欢迎安装 XT-cursor用量小工具

本程序在桌面显示 CPU、内存和 Cursor 模型用量，并提供顶栏与桌面监控面板。

安装说明
1. 默认安装到当前用户目录，一般不需要管理员权限。若你选择 Program Files 等受保护目录，请用管理员身份运行本安装程序。
2. 安装后可在开始菜单、设置 → 应用 中找到「XT-cursor用量小工具」。
3. 卸载请走 设置 → 应用，或运行安装目录里的「卸载.exe」。卸载会删除程序文件、开始菜单和桌面快捷方式、注册表项，以及 %APPDATA%\DeskHUD 中的 Cookie 与设置，不留残留。
4. 首次使用：在软件里点 Cookie，粘贴 Cursor 的 WorkosCursorSessionToken（可整段或只贴值），仅保存在本机。
5. 关闭主窗口后程序仍在托盘运行；右键托盘图标可退出。
6. 绿色版无需安装，双击即可使用，不会写入系统应用列表。

请点击「下一步」选择安装位置。`

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	if p == nil {
		p, _ = syscall.UTF16PtrFromString("?")
	}
	return p
}

func alert(title, s string) {
	owner := hwndMain
	procMessageBoxW.Call(owner, uintptr(unsafe.Pointer(utf16Ptr(s))), uintptr(unsafe.Pointer(utf16Ptr(title))), mbOK|mbIconError)
}

func infoBox(title, s string) {
	procMessageBoxW.Call(hwndMain, uintptr(unsafe.Pointer(utf16Ptr(s))), uintptr(unsafe.Pointer(utf16Ptr(title))), mbOK|mbIconInfo)
}

func exePath() string {
	if p, err := os.Executable(); err == nil && p != "" {
		return p
	}
	var buf [32768]uint16
	n, _, _ := procGetModuleFileNameW.Call(hInst, uintptr(unsafe.Pointer(&buf[0])), 32768)
	if n == 0 {
		n, _, _ = procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), 32768)
	}
	if n == 0 {
		return os.Args[0]
	}
	return syscall.UTF16ToString(buf[:n])
}

func loadAppIcon() uintptr {
	if appIcon != 0 {
		return appIcon
	}
	if hInst != 0 {
		if ico, _, _ := procLoadIconW.Call(hInst, uintptr(unsafe.Pointer(utf16Ptr("APP")))); ico != 0 {
			appIcon = ico
			return ico
		}
		if ico, _, _ := procLoadIconW.Call(hInst, 1); ico != 0 {
			appIcon = ico
			return ico
		}
	}
	path := exePath()
	if path != "" {
		u, _ := syscall.UTF16FromString(path)
		var large, small uintptr
		procExtractIconExW.Call(uintptr(unsafe.Pointer(&u[0])), 0, uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), 1)
		if large != 0 {
			appIcon = large
			return large
		}
		if small != 0 {
			appIcon = small
			return small
		}
	}
	ico, _, _ := procLoadIconW.Call(0, idiApplication)
	appIcon = ico
	return ico
}

func applyWindowIcon(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	ico := loadAppIcon()
	if ico == 0 {
		return
	}
	procSendMessageW.Call(hwnd, wmSetIcon, 1, ico)
	procSendMessageW.Call(hwnd, wmSetIcon, 0, ico)
}

func mkFont(h int32, weight int, name string) uintptr {
	hh := uint32(int32(h))
	f, _, _ := procCreateFontW.Call(uintptr(hh), 0, 0, 0, uintptr(weight), 0, 0, 0, defaultCharset, 0, 0, clearTypeQual, 2, uintptr(unsafe.Pointer(utf16Ptr(name))))
	return f
}

func selBrush(hdc uintptr, color uint32) (oldB, oldP, br, pe uintptr) {
	br, _, _ = procCreateSolidBrush.Call(uintptr(color))
	pe, _, _ = procCreatePen.Call(psSolid, 1, uintptr(color))
	oldB, _, _ = procSelectObject.Call(hdc, br)
	oldP, _, _ = procSelectObject.Call(hdc, pe)
	return
}

func unsel(hdc, oldB, oldP, br, pe uintptr) {
	procSelectObject.Call(hdc, oldB)
	procSelectObject.Call(hdc, oldP)
	procDeleteObject.Call(br)
	procDeleteObject.Call(pe)
}

func fillRound(hdc uintptr, x, y, w, h, r int32, color uint32) {
	ob, op, br, pe := selBrush(hdc, color)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h), uintptr(r), uintptr(r))
	unsel(hdc, ob, op, br, pe)
}

func fillRectC(hdc uintptr, x, y, w, h int32, color uint32) {
	ob, op, br, pe := selBrush(hdc, color)
	procRectangle.Call(hdc, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h))
	unsel(hdc, ob, op, br, pe)
}

func drawStr(hdc uintptr, x, y, w, h int32, s string, color uint32, fnt uintptr, flags uint32) {
	if fnt != 0 {
		procSelectObject.Call(hdc, fnt)
	}
	procSetBkMode.Call(hdc, transparent)
	procSetTextColor.Call(hdc, uintptr(color))
	u, _ := syscall.UTF16FromString(s)
	rc := rect{x, y, x + w, y + h}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&rc)), uintptr(flags|dtSingle|dtNoPrefix))
}

func drawWrap(hdc uintptr, x, y, w, h int32, s string, color uint32, fnt uintptr) {
	if fnt != 0 {
		procSelectObject.Call(hdc, fnt)
	}
	procSetBkMode.Call(hdc, transparent)
	procSetTextColor.Call(hdc, uintptr(color))
	u, _ := syscall.UTF16FromString(s)
	rc := rect{x, y, x + w, y + h}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&rc)), uintptr(dtWordBreak|dtNoPrefix|dtLeft))
}

func addHit(id int, x, y, w, h int32) {
	hits = append(hits, hit{id, rect{x, y, x + w, y + h}})
}

func hitTest(x, y int32) int {
	for _, h := range hits {
		if x >= h.r.left && x < h.r.right && y >= h.r.top && y < h.r.bottom {
			return h.id
		}
	}
	return 0
}

func invalidate() {
	if hwndMain != 0 {
		procInvalidateRect.Call(hwndMain, 0, 1)
	}
}

func defaultInstallDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if up := os.Getenv("USERPROFILE"); up != "" {
			base = filepath.Join(up, "AppData", "Local")
		} else {
			base = `C:\Users\Public`
		}
	}
	return filepath.Join(base, "Programs", appTitle)
}

func startMenuLnk() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", appTitle+".lnk")
}

func desktopLnk() string {
	return filepath.Join(os.Getenv("USERPROFILE"), "Desktop", appTitle+".lnk")
}

func appDataDir() string {
	a := os.Getenv("APPDATA")
	if a == "" {
		return ""
	}
	return filepath.Join(a, "DeskHUD")
}

func getEditText() string {
	if hwndEdit == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLenW.Call(hwndEdit)
	buf := make([]uint16, n+2)
	procGetWindowTextW.Call(hwndEdit, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func setEditText(s string) {
	if hwndEdit == 0 {
		return
	}
	procSetWindowTextW.Call(hwndEdit, uintptr(unsafe.Pointer(utf16Ptr(s))))
}

func showEdit(on bool) {
	if hwndEdit == 0 {
		return
	}
	cmd := uintptr(swHide)
	if on {
		cmd = swShow
	}
	procShowWindow.Call(hwndEdit, cmd)
}

func layoutEdit() {
	if hwndEdit == 0 {
		return
	}
	procSetWindowPos.Call(hwndEdit, 0, 28, 168, 430, 26, swpNoZOrder|swpNoActivate)
}

func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

var (
	clsidShellLink = guid{0x00021401, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIShellLinkW = guid{0x000214F9, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIPersistFile = guid{0x0000010b, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

func createShortcutCOM(lnk, target, workdir string) error {
	var unk uintptr
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, 1, uintptr(unsafe.Pointer(&iidIShellLinkW)), uintptr(unsafe.Pointer(&unk)))
	if hr != 0 || unk == 0 {
		return fmt.Errorf("无法创建快捷方式")
	}
	vt := *(**[32]uintptr)(unsafe.Pointer(unk))
	syscall.SyscallN(vt[20], unk, uintptr(unsafe.Pointer(utf16Ptr(target))))
	syscall.SyscallN(vt[9], unk, uintptr(unsafe.Pointer(utf16Ptr(workdir))))
	syscall.SyscallN(vt[17], unk, uintptr(unsafe.Pointer(utf16Ptr(target))), 0)
	var pf uintptr
	syscall.SyscallN(vt[0], unk, uintptr(unsafe.Pointer(&iidIPersistFile)), uintptr(unsafe.Pointer(&pf)))
	if pf == 0 {
		syscall.SyscallN(vt[2], unk)
		return fmt.Errorf("无法保存快捷方式")
	}
	pvt := *(**[8]uintptr)(unsafe.Pointer(pf))
	hr, _, _ = syscall.SyscallN(pvt[6], pf, uintptr(unsafe.Pointer(utf16Ptr(lnk))), 1)
	syscall.SyscallN(pvt[2], pf)
	syscall.SyscallN(vt[2], unk)
	if hr != 0 {
		return fmt.Errorf("保存快捷方式失败")
	}
	return nil
}

func createShortcut(lnk, target, workdir string) error {
	if err := os.MkdirAll(filepath.Dir(lnk), 0755); err != nil {
		return err
	}
	if err := createShortcutCOM(lnk, target, workdir); err == nil {
		return nil
	}
	script := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut(%s); $s.TargetPath = %s; $s.WorkingDirectory = %s; $s.IconLocation = %s; $s.Save()",
		psQuote(lnk), psQuote(target), psQuote(workdir), psQuote(target+",0"),
	)
	return hiddenCmd("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
}

func setRegSZ(key uintptr, name, val string) {
	u, _ := syscall.UTF16FromString(val)
	procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(utf16Ptr(name))), 0, regSZ, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)*2))
}

func setRegDW(key uintptr, name string, v uint32) {
	procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(utf16Ptr(name))), 0, regDWORD, uintptr(unsafe.Pointer(&v)), 4)
}

func writeUninstallReg(dir string) error {
	var hkey uintptr
	var disp uint32
	r, _, _ := procRegCreateKeyExW.Call(hkeyCU, uintptr(unsafe.Pointer(utf16Ptr(appRegKey))), 0, 0, 0, keyWrite, 0, uintptr(unsafe.Pointer(&hkey)), uintptr(unsafe.Pointer(&disp)))
	if r != 0 || hkey == 0 {
		return fmt.Errorf("无法写入卸载注册表")
	}
	defer procRegCloseKey.Call(hkey)
	app := filepath.Join(dir, appExeName)
	un := filepath.Join(dir, uninstName)
	setRegSZ(hkey, "DisplayName", appTitle)
	setRegSZ(hkey, "DisplayIcon", app+",0")
	setRegSZ(hkey, "Publisher", publisher)
	setRegSZ(hkey, "InstallLocation", dir)
	setRegSZ(hkey, "UninstallString", `"`+un+`" --uninstall`)
	setRegSZ(hkey, "DisplayVersion", appVersion)
	setRegDW(hkey, "NoModify", 1)
	setRegDW(hkey, "NoRepair", 1)
	kb := uint32((len(payload) + 1023) / 1024)
	if fi, err := os.Stat(un); err == nil {
		kb += uint32((fi.Size() + 1023) / 1024)
	} else if fi, err := os.Stat(exePath()); err == nil {
		kb += uint32((fi.Size() + 1023) / 1024)
	}
	setRegDW(hkey, "EstimatedSize", kb)
	return nil
}

func queryRegSZ(key uintptr, name string) string {
	var typ, n uint32
	buf := make([]uint16, 1024)
	n = uint32(len(buf) * 2)
	r, _, _ := procRegQueryValueExW.Call(key, uintptr(unsafe.Pointer(utf16Ptr(name))), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r != 0 {
		return ""
	}
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func detectInstall() {
	hasExisting = false
	existingDir, existingVer = "", ""
	const keyRead = 0x20019
	var hkey uintptr
	r, _, _ := procRegOpenKeyExW.Call(hkeyCU, uintptr(unsafe.Pointer(utf16Ptr(appRegKey))), 0, keyRead, uintptr(unsafe.Pointer(&hkey)))
	dir, ver := "", ""
	if r == 0 && hkey != 0 {
		dir = queryRegSZ(hkey, "InstallLocation")
		ver = queryRegSZ(hkey, "DisplayVersion")
		procRegCloseKey.Call(hkey)
	}
	if dir == "" {
		dir = defaultInstallDir()
	}
	exe := filepath.Join(dir, appExeName)
	if st, err := os.Stat(exe); err == nil && !st.IsDir() {
		hasExisting = true
		existingDir = dir
		if ver == "" {
			ver = "未知"
		}
		existingVer = ver
	}
}

func startUpdate() {
	if busy || existingDir == "" {
		return
	}
	installDir = existingDir
	progMsg = "准备更新…"
	beginInstall()
}

func startOverwrite() {
	if busy {
		return
	}
	page = pageFolder
	if existingDir != "" {
		setEditText(existingDir)
	}
	showEdit(true)
	layoutEdit()
	invalidate()
}

func deleteUninstallReg() {
	procRegDeleteKeyW.Call(hkeyCU, uintptr(unsafe.Pointer(utf16Ptr(appRegKey))))
}

func killApp() {
	_ = hiddenCmd("taskkill", "/F", "/IM", appExeName).Run()
}

func findAppWindow() uintptr {
	for _, cls := range []string{"DeskHUD", "DeskHUDDash", "DeskHUDPanel"} {
		h, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16Ptr(cls))), 0)
		if h != 0 {
			return h
		}
	}
	return 0
}

func appRunning() bool {
	if findAppWindow() != 0 {
		return true
	}
	out, err := hiddenCmd("tasklist", "/FI", "IMAGENAME eq "+appExeName, "/NH").CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	return strings.Contains(s, "xt-cursor") || strings.Contains(s, strings.ToLower(appExeName))
}

func stopRunningApp() bool {
	if h := findAppWindow(); h != 0 {
		procPostMessageW.Call(h, wmClose, 0, 0)
	}
	if d, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16Ptr("DeskHUDDash"))), 0); d != 0 {
		procPostMessageW.Call(d, wmClose, 0, 0)
	}
	if h, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16Ptr("DeskHUD"))), 0); h != 0 {
		procPostMessageW.Call(h, wmClose, 0, 0)
	}
	for i := 0; i < 20; i++ {
		time.Sleep(150 * time.Millisecond)
		if !appRunning() {
			return true
		}
	}
	_ = hiddenCmd("taskkill", "/IM", appExeName).Run()
	time.Sleep(400 * time.Millisecond)
	if !appRunning() {
		return true
	}
	_ = hiddenCmd("taskkill", "/F", "/IM", appExeName).Run()
	time.Sleep(400 * time.Millisecond)
	return !appRunning()
}

func beginInstall() {
	if appRunning() {
		if page != pageRunning {
			returnPage = page
		}
		if returnPage == pageProgress {
			returnPage = pageExist
		}
		page = pageRunning
		showEdit(false)
		invalidate()
		return
	}
	showEdit(false)
	page = pageProgress
	progPct = 0
	if progMsg == "" {
		progMsg = "准备安装…"
	}
	busy = true
	invalidate()
	go runInstall()
}

func doStop() {
	if busy || stopping {
		return
	}
	stopping = true
	busy = true
	invalidate()
	go func() {
		ok := stopRunningApp()
		stopping = false
		if ok {
			procPostMessageW.Call(hwndMain, wmAppStopped, 0, 0)
		} else {
			failMsg = "无法退出正在运行的程序。请先在托盘图标上右键「退出」，再回来安装。"
			procPostMessageW.Call(hwndMain, wmAppFail, 0, 0)
		}
	}()
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return strings.EqualFold(a, b)
	}
	return strings.EqualFold(aa, bb)
}

func setProgress(pct int, msg string) {
	progPct = pct
	progMsg = msg
	if hwndMain != 0 {
		procPostMessageW.Call(hwndMain, wmAppStatus, 0, 0)
	}
}

func runInstall() {
	dir := installDir
	fail := func(s string) {
		failMsg = s
		procPostMessageW.Call(hwndMain, wmAppFail, 0, 0)
	}
	if len(payload) == 0 {
		fail("安装程序损坏：缺少程序文件。")
		return
	}
	setProgress(8, "正在创建安装目录…")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fail("无法创建安装目录：\n" + dir + "\n\n" + err.Error() + "\n\n若目标是 Program Files 等受保护目录，请用管理员身份运行本安装程序。")
		return
	}
	if appRunning() {
		fail("原程序仍在运行。请先点「停止」退出后再安装。")
		return
	}
	setProgress(28, "正在写入程序文件…")
	appPath := filepath.Join(dir, appExeName)
	if err := os.WriteFile(appPath, payload, 0644); err != nil {
		fail("无法写入程序文件：\n" + appPath + "\n\n" + err.Error() + "\n\n若目标是 Program Files 等受保护目录，请用管理员身份运行本安装程序。")
		return
	}
	setProgress(50, "正在写入卸载程序…")
	self := exePath()
	selfData, err := os.ReadFile(self)
	if err != nil {
		fail("无法读取安装程序自身：\n" + err.Error())
		return
	}
	unPath := filepath.Join(dir, uninstName)
	if err := os.WriteFile(unPath, selfData, 0644); err != nil {
		fail("无法写入卸载程序：\n" + unPath + "\n\n" + err.Error() + "\n\n若目标是 Program Files 等受保护目录，请用管理员身份运行本安装程序。")
		return
	}
	setProgress(68, "正在创建开始菜单快捷方式…")
	if err := createShortcut(startMenuLnk(), appPath, dir); err != nil {
		fail("无法创建开始菜单快捷方式：\n" + err.Error())
		return
	}
	if wantDesktop {
		setProgress(82, "正在创建桌面快捷方式…")
		if err := createShortcut(desktopLnk(), appPath, dir); err != nil {
			fail("无法创建桌面快捷方式：\n" + err.Error())
			return
		}
	} else {
		setProgress(82, "已跳过桌面快捷方式")
	}
	setProgress(92, "正在写入卸载信息…")
	if err := writeUninstallReg(dir); err != nil {
		fail(err.Error())
		return
	}
	setProgress(100, "安装完成")
	procPostMessageW.Call(hwndMain, wmAppDone, 0, 0)
}

func runUninstall() {
	dir := installDir
	if dir == "" {
		dir = filepath.Dir(exePath())
		installDir = dir
	}
	setProgress(10, "正在结束运行中的程序…")
	killApp()
	setProgress(28, "正在删除快捷方式…")
	_ = os.Remove(startMenuLnk())
	_ = os.Remove(desktopLnk())
	setProgress(46, "正在删除本机设置…")
	if d := appDataDir(); d != "" {
		_ = os.RemoveAll(d)
	}
	setProgress(64, "正在删除注册表项…")
	deleteUninstallReg()
	setProgress(80, "正在删除程序文件…")
	self := exePath()
	if ents, err := os.ReadDir(dir); err == nil {
		for _, e := range ents {
			p := filepath.Join(dir, e.Name())
			if samePath(p, self) {
				continue
			}
			_ = os.RemoveAll(p)
		}
	}
	setProgress(94, "正在移除安装目录…")
	cmd := exec.Command("cmd", "/C", `ping 127.0.0.1 -n 2 & rd /s /q "`+dir+`"`)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | detachedProcess}
	_ = cmd.Start()
	setProgress(100, "卸载完成")
	procPostMessageW.Call(hwndMain, wmAppDone, 0, 0)
}

func browseFolder() {
	var display [260]uint16
	bi := browseInfoW{
		hwndOwner:      hwndMain,
		pszDisplayName: &display[0],
		lpszTitle:      utf16Ptr("选择安装文件夹"),
		ulFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle,
	}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return
	}
	defer procCoTaskMemFree.Call(pidl)
	var path [260]uint16
	r, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 {
		return
	}
	s := syscall.UTF16ToString(path[:])
	if s == "" {
		return
	}
	if !strings.EqualFold(filepath.Base(s), appTitle) {
		s = filepath.Join(s, appTitle)
	}
	setEditText(s)
}

func launchApp() {
	exe := filepath.Join(installDir, appExeName)
	cmd := exec.Command(exe)
	cmd.Dir = installDir
	_ = cmd.Start()
}

func drawBtn(hdc uintptr, x, y, w, h int32, label string, id int, enabled, primary bool) {
	bg, fg := uint32(colInner), uint32(colMute)
	if enabled {
		if primary {
			bg, fg = colCyan, colInk
		} else {
			bg, fg = colBtn, colWhite
		}
	}
	fillRound(hdc, x, y, w, h, 8, bg)
	drawStr(hdc, x, y, w, h, label, fg, fontBody, dtCenter|dtVCenter)
	if enabled {
		addHit(id, x, y, w, h)
	}
}

func paintHeader(hdc uintptr, cw int32, title, sub string) {
	fillRectC(hdc, 0, 0, cw, 56, colCard)
	fillRectC(hdc, 0, 0, 6, 56, colCyan)
	drawStr(hdc, 20, 8, cw-40, 24, title, colWhite, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, 20, 32, cw-40, 18, sub, colMute, fontSmall, dtLeft|dtVCenter)
}

func paintFooter(hdc uintptr, cw, ch int32) {
	fillRectC(hdc, 0, ch-56, cw, 56, colCard)
	y := ch - 44
	bw, bh, gap := int32(88), int32(32), int32(10)
	right := cw - 20
	place := func(label string, id int, enabled, primary bool) {
		right -= bw
		drawBtn(hdc, right, y, bw, bh, label, id, enabled, primary)
		right -= gap
	}
	if uninstallMode {
		switch page {
		case pageUnAsk:
			place("取消", hitCancel, !busy, false)
			place("卸载", hitUninst, !busy, true)
		case pageUnProg:
			place("取消", hitCancel, false, false)
		case pageUnDone:
			place("完成", hitFinish, true, true)
		}
		return
	}
	switch page {
	case pageRunning:
		place("取消", hitCancel, !busy && !stopping, false)
		place("停止", hitStop, !busy && !stopping, true)
	case pageExist:
		place("取消", hitCancel, !busy, false)
		bw = 108
		place("覆盖安装", hitOverwrite, !busy, false)
		bw = 88
		place("更新", hitUpdate, !busy, true)
	case pageIntro:
		place("取消", hitCancel, !busy, false)
		place("下一步", hitNext, !busy, true)
	case pageFolder:
		place("取消", hitCancel, !busy, false)
		place("下一步", hitNext, !busy, true)
		place("上一步", hitBack, !busy, false)
	case pageProgress:
		place("取消", hitCancel, false, false)
	case pageDone:
		place("完成", hitFinish, true, true)
	}
}

func paintContent(hdc uintptr, cw, ch int32) {
	x, y, w := int32(24), int32(68), cw-48
	h := ch - 56 - 68
	if uninstallMode {
		switch page {
		case pageUnAsk:
			drawStr(hdc, x, y, w, 26, "卸载确认", colWhite, fontTitle, dtLeft|dtVCenter)
			drawWrap(hdc, x, y+36, w, h-40, "确定要卸载 XT-cursor用量小工具吗？将删除程序、快捷方式和本机设置。", colWhite, fontBody)
		case pageUnProg:
			drawStr(hdc, x, y, w, 26, "正在卸载", colWhite, fontTitle, dtLeft|dtVCenter)
			paintBar(hdc, x, y+50, w, 16)
			drawStr(hdc, x, y+76, w, 22, progMsg, colMute, fontSmall, dtLeft|dtVCenter)
		case pageUnDone:
			drawStr(hdc, x, y, w, 32, "卸载完成", colWhite, fontBig, dtLeft|dtVCenter)
			drawWrap(hdc, x, y+48, w, 80, "卸载完成，无残留。", colWhite, fontBody)
		}
		return
	}
	switch page {
	case pageRunning:
		drawStr(hdc, x, y, w, 28, "原程序正在运行", colWhite, fontTitle, dtLeft|dtVCenter)
		txt := "检测到 XT-cursor用量小工具 正在运行，无法直接覆盖文件。\n\n请先点击「停止」退出已安装的程序，然后再继续安装。\nCookie 和设置会保留。"
		if stopping {
			txt = "正在退出已安装的程序…"
		}
		drawWrap(hdc, x, y+36, w, h-40, txt, colWhite, fontBody)
	case pageExist:
		drawStr(hdc, x, y, w, 28, "检测到已安装的程序", colWhite, fontTitle, dtLeft|dtVCenter)
		msg := "本机已安装 XT-cursor用量小工具。\n\n已安装版本：" + existingVer + "\n当前安装包：" + appVersion + "\n安装位置：\n" + existingDir + "\n\n更新：覆盖到原位置，保留 Cookie 和设置。\n覆盖安装：可改安装目录后再写入。\n取消：不改动现有程序。"
		drawWrap(hdc, x, y+36, w, h-40, msg, colWhite, fontBody)
	case pageIntro:
		drawWrap(hdc, x, y, w, h, introText, colWhite, fontSmall)
	case pageFolder:
		drawStr(hdc, x, y, w, 24, "选择安装文件夹", colWhite, fontTitle, dtLeft|dtVCenter)
		drawWrap(hdc, x, y+30, w, 40, "程序将安装到下面的文件夹。可直接编辑路径，或点击「浏览」选择。", colMute, fontSmall)
		drawBtn(hdc, 468, 166, 88, 30, "浏览", hitBrowse, true, false)
		bx, by := int32(24), int32(214)
		bg := uint32(colTrack)
		if wantDesktop {
			bg = colCyan
		}
		fillRound(hdc, bx, by, 22, 22, 6, bg)
		if wantDesktop {
			drawStr(hdc, bx, by, 22, 22, "✓", colInk, fontBody, dtCenter|dtVCenter)
		}
		drawStr(hdc, bx+32, by, 360, 22, "创建桌面快捷方式", colWhite, fontBody, dtLeft|dtVCenter)
		addHit(hitDeskCB, bx, by, 280, 22)
		drawStr(hdc, x, by+32, w, 20, "开始菜单快捷方式将始终创建。", colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, x, by+56, w, 36, "默认安装到当前用户目录，一般不需要管理员权限。", colMute, fontSmall, dtLeft)
	case pageProgress:
		drawStr(hdc, x, y, w, 26, "正在安装", colWhite, fontTitle, dtLeft|dtVCenter)
		paintBar(hdc, x, y+50, w, 16)
		drawStr(hdc, x, y+76, w, 22, progMsg, colMute, fontSmall, dtLeft|dtVCenter)
		if installDir != "" {
			drawStr(hdc, x, y+104, w, 20, installDir, colMute, fontSmall, dtLeft)
		}
	case pageDone:
		drawStr(hdc, x, y, w, 32, "安装完成", colWhite, fontBig, dtLeft|dtVCenter)
		drawWrap(hdc, x, y+48, w, 80, "已安装到：\n"+installDir+"\n\n可在开始菜单、设置 → 应用 中找到本程序。点击「完成」启动 XT-cursor用量小工具。", colWhite, fontBody)
	}
}

func paintBar(hdc uintptr, x, y, w, h int32) {
	fillRound(hdc, x, y, w, h, h, colTrack)
	pct := progPct
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fw := w * int32(pct) / 100
	if fw < h && pct > 0 {
		fw = h
	}
	if pct > 0 {
		fillRound(hdc, x, y, fw, h, h, colCyan)
	}
	drawStr(hdc, x, y, w, h, fmt.Sprintf("%d%%", pct), colWhite, fontSmall, dtCenter|dtVCenter)
}

func paintWizard(hwnd, hdc uintptr) {
	var rc rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	cw, ch := rc.right, rc.bottom
	mem, _, _ := procCreateCompatibleDC.Call(hdc)
	if mem == 0 {
		return
	}
	bmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(cw), uintptr(ch))
	if bmp == 0 {
		procDeleteDC.Call(mem)
		return
	}
	old, _, _ := procSelectObject.Call(mem, bmp)
	fillRectC(mem, 0, 0, cw, ch, colBoard)
	hits = hits[:0]
	if uninstallMode {
		paintHeader(mem, cw, uninstTitle, "卸载向导")
	} else {
		subs := []string{"安装说明", "选择安装文件夹", "安装进度", "完成", "检测到已安装", "程序正在运行"}
		sub := "安装向导"
		if page >= 0 && page < len(subs) {
			sub = subs[page]
		}
		paintHeader(mem, cw, setupTitle+"  v"+appVersion, sub)
	}
	paintContent(mem, cw, ch)
	paintFooter(mem, cw, ch)
	procBitBlt.Call(hdc, 0, 0, uintptr(cw), uintptr(ch), mem, 0, 0, srcCopy)
	procSelectObject.Call(mem, old)
	procDeleteObject.Call(bmp)
	procDeleteDC.Call(mem)
}

func goNext() {
	if uninstallMode || busy {
		return
	}
	switch page {
	case pageIntro:
		page = pageFolder
		showEdit(true)
		layoutEdit()
		invalidate()
	case pageFolder:
		p := getEditText()
		if p == "" {
			alert(setupTitle, "请选择安装文件夹。")
			return
		}
		installDir = p
		progMsg = "准备安装…"
		beginInstall()
	}
}

func goBack() {
	if uninstallMode || busy {
		return
	}
	if page == pageFolder {
		if hasExisting {
			page = pageExist
		} else {
			page = pageIntro
		}
		showEdit(false)
		invalidate()
	}
}

func goCancel() {
	if busy {
		return
	}
	if page == pageRunning {
		if returnPage == pageFolder {
			page = pageFolder
			showEdit(true)
			layoutEdit()
		} else {
			page = pageExist
			showEdit(false)
		}
		invalidate()
		return
	}
	procDestroyWindow.Call(hwndMain)
}

func goFinish() {
	if uninstallMode {
		procDestroyWindow.Call(hwndMain)
		return
	}
	launchApp()
	procDestroyWindow.Call(hwndMain)
}

func startUninstall() {
	if busy {
		return
	}
	installDir = filepath.Dir(exePath())
	page = pageUnProg
	progPct = 0
	progMsg = "准备卸载…"
	busy = true
	invalidate()
	go runUninstall()
}

func onClick(id int) {
	switch id {
	case hitNext:
		goNext()
	case hitBack:
		goBack()
	case hitCancel:
		goCancel()
	case hitBrowse:
		browseFolder()
	case hitDeskCB:
		wantDesktop = !wantDesktop
		invalidate()
	case hitFinish:
		goFinish()
	case hitUninst:
		startUninstall()
	case hitUpdate:
		startUpdate()
	case hitOverwrite:
		startOverwrite()
	case hitStop:
		doStop()
	}
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmEraseBk:
		return 1
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		paintWizard(hwnd, hdc)
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmCtlColorEdit:
		procSetTextColor.Call(wParam, colWhite)
		procSetBkColor.Call(wParam, colCard)
		procSetBkMode.Call(wParam, opaque)
		return cardBrush
	case wmSetCursor:
		if arrowCursor != 0 {
			procSetCursor.Call(arrowCursor)
		}
		return 1
	case wmLButtonUp:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		if id := hitTest(x, y); id != 0 {
			onClick(id)
		}
		return 0
	case wmAppStatus:
		invalidate()
		return 0
	case wmAppStopped:
		busy = false
		stopping = false
		beginInstall()
		return 0
	case wmAppDone:
		busy = false
		if uninstallMode {
			page = pageUnDone
		} else {
			page = pageDone
		}
		invalidate()
		return 0
	case wmAppFail:
		busy = false
		stopping = false
		alert(setupTitle, failMsg)
		if page == pageRunning {
			invalidate()
			return 0
		}
		page = pageFolder
		showEdit(true)
		layoutEdit()
		invalidate()
		return 0
	case wmClose:
		if busy {
			return 0
		}
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		hwndMain = 0
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func isUninstallMode() bool {
	for _, a := range os.Args[1:] {
		if a == "--uninstall" {
			return true
		}
	}
	base := filepath.Base(os.Args[0])
	if p, err := os.Executable(); err == nil && p != "" {
		base = filepath.Base(p)
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return name == "卸载"
}

func main() {
	runtime.LockOSThread()
	defer func() {
		if r := recover(); r != nil {
			alert(setupTitle, fmt.Sprintf("启动失败: %v", r))
		}
	}()
	uninstallMode = isUninstallMode()
	if !uninstallMode {
		detectInstall()
		if hasExisting {
			page = pageExist
		}
	}
	procCoInitializeEx.Call(0, coInitApartment)
	procSetProcessDPIAware.Call()
	hInst, _, _ = procGetModuleHandleW.Call(0)
	arrowCursor, _, _ = procLoadCursorW.Call(0, idcArrow)
	boardBrush, _, _ = procCreateSolidBrush.Call(colBoard)
	cardBrush, _, _ = procCreateSolidBrush.Call(colCard)
	fontTitle = mkFont(-18, 600, "宋体")
	fontBody = mkFont(-15, 400, "宋体")
	fontSmall = mkFont(-13, 400, "宋体")
	fontBig = mkFont(-22, 700, "宋体")

	ico := loadAppIcon()
	className := "XTSetupWizard"
	wc := wndClassEx{
		size:       uint32(unsafe.Sizeof(wndClassEx{})),
		wndProc:    syscall.NewCallback(wndProc),
		instance:   syscall.Handle(hInst),
		icon:       syscall.Handle(ico),
		cursor:     syscall.Handle(arrowCursor),
		background: syscall.Handle(boardBrush),
		className:  utf16Ptr(className),
		iconSm:     syscall.Handle(ico),
	}
	if a, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); a == 0 {
		alert(setupTitle, "窗口类注册失败")
		return
	}
	cx, _, _ := procGetSystemMetrics.Call(smCxScreen)
	cy, _, _ := procGetSystemMetrics.Call(smCyScreen)
	x := (int32(cx) - winW) / 2
	y := (int32(cy) - winH) / 2
	title := setupTitle
	if uninstallMode {
		title = uninstTitle
	}
	style := uintptr(wsOverlapped | wsCaption | wsSysmenu | wsMinimizeBox | wsVisible | wsClipChildren)
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(className))), uintptr(unsafe.Pointer(utf16Ptr(title))),
		style, uintptr(x), uintptr(y), uintptr(winW), uintptr(winH), 0, 0, hInst, 0)
	if hwnd == 0 {
		alert(setupTitle, "窗口创建失败: "+err.Error())
		return
	}
	hwndMain = hwnd
	applyWindowIcon(hwnd)

	if !uninstallMode {
		hwndE, _, _ := procCreateWindowExW.Call(wsExClientEdge, uintptr(unsafe.Pointer(utf16Ptr("EDIT"))), uintptr(unsafe.Pointer(utf16Ptr(defaultInstallDir()))),
			wsChild|esLeft|esAutoHScroll|wsTabstop, 28, 168, 430, 26, hwnd, 100, hInst, 0)
		hwndEdit = hwndE
		if hwndEdit != 0 && fontBody != 0 {
			procSendMessageW.Call(hwndEdit, wmSetFont, fontBody, 1)
		}
		showEdit(false)
	} else {
		installDir = filepath.Dir(exePath())
	}

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	for _, f := range []uintptr{fontTitle, fontBody, fontSmall, fontBig, boardBrush, cardBrush} {
		if f != 0 {
			procDeleteObject.Call(f)
		}
	}
}
