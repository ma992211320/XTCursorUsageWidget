package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	wsPopup        = 0x80000000
	wsVisible      = 0x10000000
	wsCaption      = 0x00C00000
	wsSysmenu      = 0x00080000
	wsMinimizeBox  = 0x00020000
	wsMaximizeBox  = 0x00010000
	wsThickframe   = 0x00040000
	wsOverlapped   = 0x00000000
	wsClipChildren = 0x02000000
	wsExLayered     = 0x00080000
	wsExTopmost     = 0x00000008
	wsExNoActivate  = 0x08000000
	wsExToolwindow  = 0x00000080
	wsExTransparent = 0x00000020
	wmDestroy      = 0x0002
	wmClose        = 0x0010
	wmSize         = 0x0005
	wmGetMinMaxInfo = 0x0024
	wmNCLButtonDown = 0x00A1
	wmNCCalcSize   = 0x0083
	wmNCHitTest    = 0x0084
	wmPaint        = 0x000F
	wmRButtonUp    = 0x0205
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmMouseMove    = 0x0200
	wmSetCursor    = 0x0020
	wmCommand      = 0x0111
	wmApp          = 0x8000
	wmAppRefresh   = wmApp + 1
	wmTray         = wmApp + 2
	wmLButtonDblClk = 0x0203
	wmContextMenu  = 0x007B
	wmNull         = 0x0000
	swShow         = 5
	swShowNA       = 8
	swHide         = 0
	swMinimize     = 6
	swMaximize     = 3
	swRestore      = 9
	htClient       = 1
	htCaption      = 2
	htLeft         = 10
	htRight        = 11
	htTop          = 12
	htTopLeft      = 13
	htTopRight     = 14
	htBottom       = 15
	htBottomLeft   = 16
	htBottomRight  = 17
	sizeMinimized  = 1
	frameHit       = 8
	dwmBorderColor = 34
	dwmCaptionColor = 35
	capH           = 36
	tpmRightbutton = 0x0002
	tpmBottomalign = 0x0020
	tpmRightalign  = 0x0008
	mfString       = 0x0000
	idQuit         = 1001
	idOpen         = 1002
	lwaColorKey    = 0x00000001
	lwaAlpha       = 0x00000002
	colorKey       = 0x00FF00FF
	transparent    = 1
	antiAliasQual  = 4
	clearTypeQual  = 5
	defaultCharset = 134
	mbOK           = 0
	mbIconError    = 0x10
	smCxScreen     = 0
	smCyScreen     = 1
	swpNoActivate  = 0x0010
	swpNoSize      = 0x0001
	hwndTopmost    = ^uintptr(0)
	hwndNoTopmost  = ^uintptr(1)
	idcArrow       = 32512
	idiApplication = 32512
	nimAdd, nimDelete = 0, 2
	nifMessage, nifIcon, nifTip = 0x1, 0x2, 0x4
	ccRGBInit      = 1
	ccFullOpen     = 2
	ccAnyColor     = 0x100
	dtLeft         = 0
	dtCenter       = 1
	dtRight        = 2
	dtVCenter      = 4
	dtSingle       = 0x20
	psSolid        = 0
	nullPen        = 8

	// BGR（对齐网站：底 #070B14 / 卡片 #121826 / 描边 #243044 / 青 #4AD5E5）
	colBoard  = 0x00140B07
	colCard   = 0x00261812
	colInner  = 0x00201A14
	colLine   = 0x00443024
	colCyan   = 0x00E5D54A
	colLilac  = 0x00FDB5C4
	colOrange = 0x0048A0F0
	colGreen  = 0x008ACF3D
	colWhite  = 0x00FBF3EE
	colMute   = 0x00B09B8B
	colTrack  = 0x002A2018
	colInk    = 0x00381F05
	nullBrush = 5
	diNormal  = 0x0003
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	psapi    = syscall.NewLazyDLL("psapi.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	msimg32  = syscall.NewLazyDLL("msimg32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procSetCapture           = user32.NewProc("SetCapture")
	procReleaseCapture       = user32.NewProc("ReleaseCapture")
	procCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	procAppendMenuW          = user32.NewProc("AppendMenuW")
	procTrackPopupMenu       = user32.NewProc("TrackPopupMenu")
	procDestroyMenu          = user32.NewProc("DestroyMenu")
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procSetCursor            = user32.NewProc("SetCursor")
	procSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procSetLayeredWindowAttr = user32.NewProc("SetLayeredWindowAttributes")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procBeginPaint           = user32.NewProc("BeginPaint")
	procEndPaint             = user32.NewProc("EndPaint")
	procFillRect             = user32.NewProc("FillRect")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procInvalidateRect       = user32.NewProc("InvalidateRect")
	procDrawTextW            = user32.NewProc("DrawTextW")
	procFindWindowW          = user32.NewProc("FindWindowW")
	procFindWindowExW        = user32.NewProc("FindWindowExW")
	procEnumWindows          = user32.NewProc("EnumWindows")
	procSendMessageTimeoutW  = user32.NewProc("SendMessageTimeoutW")
	procSetParent            = user32.NewProc("SetParent")
	procGetParent            = user32.NewProc("GetParent")
	procGetClassNameW        = user32.NewProc("GetClassNameW")
	procScreenToClient       = user32.NewProc("ScreenToClient")

	procCreateSolidBrush     = gdi32.NewProc("CreateSolidBrush")
	procCreatePen            = gdi32.NewProc("CreatePen")
	procDeleteObject         = gdi32.NewProc("DeleteObject")
	procCreateFontW          = gdi32.NewProc("CreateFontW")
	procSelectObject         = gdi32.NewProc("SelectObject")
	procSetBkMode            = gdi32.NewProc("SetBkMode")
	procSetTextColor         = gdi32.NewProc("SetTextColor")
	procTextOutW             = gdi32.NewProc("TextOutW")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
	procRoundRect            = gdi32.NewProc("RoundRect")
	procRectangle            = gdi32.NewProc("Rectangle")
	procEllipse              = gdi32.NewProc("Ellipse")
	procPie                  = gdi32.NewProc("Pie")
	procGetStockObject       = gdi32.NewProc("GetStockObject")
	procCreateCompatibleDC   = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procCreateDIBSection     = gdi32.NewProc("CreateDIBSection")
	procBitBlt               = gdi32.NewProc("BitBlt")
	procDeleteDC             = gdi32.NewProc("DeleteDC")
	procMoveToEx             = gdi32.NewProc("MoveToEx")
	procLineTo               = gdi32.NewProc("LineTo")
	procSetViewportOrgEx     = gdi32.NewProc("SetViewportOrgEx")
	procAlphaBlend           = msimg32.NewProc("AlphaBlend")
	procDrawIconEx           = user32.NewProc("DrawIconEx")
	procIsZoomed             = user32.NewProc("IsZoomed")

	procOpenClipboard            = user32.NewProc("OpenClipboard")
	procCloseClipboard           = user32.NewProc("CloseClipboard")
	procGetClipboardData         = user32.NewProc("GetClipboardData")
	procGetSystemTimes           = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx     = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	procGlobalLock               = kernel32.NewProc("GlobalLock")
	procGlobalUnlock             = kernel32.NewProc("GlobalUnlock")
	procChooseColorW             = comdlg32.NewProc("ChooseColorW")
	procGetPerformanceInfo       = psapi.NewProc("GetPerformanceInfo")
	procNtQuerySystemInformation = ntdll.NewProc("NtQuerySystemInformation")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procIsIconic                 = user32.NewProc("IsIconic")
	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
	procExtractIconExW           = shell32.NewProc("ExtractIconExW")
	procGetModuleFileNameW       = kernel32.NewProc("GetModuleFileNameW")
	procRegCreateKeyExW          = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueExW           = advapi32.NewProc("RegSetValueExW")
	procRegCloseKey              = advapi32.NewProc("RegCloseKey")
	procRegDeleteKeyW            = advapi32.NewProc("RegDeleteKeyW")
	procDwmSetWindowAttribute    = dwmapi.NewProc("DwmSetWindowAttribute")
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
	hwnd    syscall.Handle
	message uint32
	wParam, lParam uintptr
	time    uint32
	pt      point
	private uint32
}
type paintStruct struct {
	hdc         syscall.Handle
	erase       int32
	rcPaint     rect
	restore, incUpdate int32
	rgbReserved [32]byte
}
type memoryStatusEx struct {
	length, memoryLoad uint32
	totalPhys, availPhys, totalPageFile, availPageFile, totalVirtual, availVirtual, availExtendedVirtual uint64
}
type filetime struct{ low, high uint32 }
type posFile struct{ X, Y int32 }
type minmaxinfo struct {
	reserved point
	maxSize  point
	maxPos   point
	minTrack point
	maxTrack point
}
type settings struct {
	ShowCPU     bool   `json:"showCpu"`
	ShowMem     bool   `json:"showMem"`
	ShowCursor  bool   `json:"showCursor"`
	ShowBar     bool   `json:"showBar"`
	ShowDesk    bool   `json:"showDesk"`
	Color       uint32 `json:"color"`
	Pinned      bool   `json:"pinned"`
	SkipVersion string `json:"skipVersion"`
	BarAlpha    int    `json:"barAlpha"`
	DeskAlpha   int    `json:"deskAlpha"`
	DashW       int    `json:"dashW"`
	DashH       int    `json:"dashH"`
}
type chooseColorW struct {
	lStructSize uint32
	_pad0       uint32
	hwndOwner, hInstance uintptr
	rgbResult   uint32
	_pad1       uint32
	lpCustColors uintptr
	flags       uint32
	_pad2       uint32
	lCustData, lpfnHook, lpTemplateName uintptr
}
type perfInfo struct {
	cb uint32
	_pad uint32
	commitTotal, commitLimit, commitPeak, physicalTotal, physicalAvailable, systemCache, kernelTotal, kernelPaged, kernelNonpaged, pageSize uintptr
	handleCount, processCount, threadCount uint32
}
type memListInfo struct {
	zeroPage, freePage, modifiedPage, modifiedNoWrite, badPage uintptr
	priority   [8]uintptr
	repurposed [8]uintptr
	modifiedPageFile uintptr
}
type notifyIconData struct {
	cbSize           uint32
	_pad0            uint32
	hwnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	_pad1            uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}
type hit struct {
	id int
	r  rect
}
type memDetail struct {
	totalGB, usedGB, availGB float64
	usedPct int
	commitUsedGB, commitLimitGB, cachedGB float64
	pagedMB, nonpagedMB, sysCacheUsedMB, sysCacheTotalMB float64
}

const (
	hudW, hudH int32 = 820, 44
	dashW, dashH int32 = 760, 700
	hitTab0, hitTab1, hitTab2, hitTab3 = 1, 2, 3, 8
	hitPin, hitCpu, hitMem, hitColor = 4, 5, 6, 7
	hitCookie, hitRefresh, hitCursor, hitQuit = 9, 10, 11, 12
	hitBar, hitDesk, hitUninstall, hitCheckUpdate = 13, 14, 15, 16
	hitBarAlpha, hitDeskAlpha                     = 17, 18
	hitCapMin, hitCapMax, hitCapClose             = 19, 20, 21
)

var (
	prevIdle, prevKernel, prevUser uint64
	cpuPercent, memLoad            int
	usedGB, totalGB                float64
	detail                         memDetail
	fontHUD, fontTitle, fontBody, fontBig, fontHuge, fontSmall uintptr
	keyBrush, boardBrush, arrowCursor, hInst uintptr
	hwndHUD, hwndDash, hwndDesk uintptr
	cfg settings
	custColors [16]uint32
	hits []hit
	page int
	pendingHUD, pendingDash, pendingDesk, pendingMove, pendingClick, pendingColor, pendingCookie bool
	hwndCookie, hwndEdit uintptr
	clickX, clickY, moveX, moveY, dragOffX, dragOffY int32
	dragging bool
	dragHWND uintptr
	pendingOpen, pendingTray, pendingUninstall bool
	trayAdded bool
	foundWorker uintptr
	foundDefView uintptr
	enumWorkerCb = syscall.NewCallback(enumDesktopWorker)
	enumDefViewCb = syscall.NewCallback(enumDefViewWorker)
	trackBarAlpha, trackDeskAlpha rect
	uiW, hitOff int32
	settingsNeedSave bool
	slideID int
	pendingSlide, pendingSlideEnd bool
)

func enumDesktopWorker(hwnd, lParam uintptr) uintptr {
	var buf [32]uint16
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 32)
	if syscall.UTF16ToString(buf[:]) != "WorkerW" {
		return 1
	}
	shell, _, _ := procFindWindowExW.Call(hwnd, 0, uintptr(unsafe.Pointer(utf16Ptr("SHELLDLL_DefView"))), 0)
	if shell == 0 {
		foundWorker = hwnd
	}
	return 1
}

func enumDefViewWorker(hwnd, lParam uintptr) uintptr {
	var buf [32]uint16
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 32)
	if syscall.UTF16ToString(buf[:]) != "WorkerW" {
		return 1
	}
	shell, _, _ := procFindWindowExW.Call(hwnd, 0, uintptr(unsafe.Pointer(utf16Ptr("SHELLDLL_DefView"))), 0)
	if shell != 0 {
		foundDefView = shell
	}
	return 1
}


func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	if p == nil {
		p, _ = syscall.UTF16PtrFromString("?")
	}
	return p
}
func alert(t, s string) {
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(s))), uintptr(unsafe.Pointer(utf16Ptr(t))), mbOK|mbIconError)
}
func appDir() string {
	a := os.Getenv("APPDATA")
	if a == "" {
		a = "."
	}
	return filepath.Join(a, "DeskHUD")
}
func clampAlphaVal(v int) int {
	if v < 30 {
		return 30
	}
	if v > 100 {
		return 100
	}
	return ((v + 5) / 10) * 10
}

func loadSettings() settings {
	s := settings{ShowCPU: true, ShowMem: true, ShowCursor: true, ShowBar: true, ShowDesk: true, Color: 0x00FFFFFF, Pinned: true, BarAlpha: 80, DeskAlpha: 70}
	b, err := os.ReadFile(filepath.Join(appDir(), "settings.json"))
	if err != nil {
		return s
	}
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	_ = json.Unmarshal(b, &s)
	if _, ok := raw["showBar"]; !ok {
		s.ShowBar = true
	}
	if _, ok := raw["showDesk"]; !ok {
		s.ShowDesk = true
	}
	_, hasBar := raw["barAlpha"]
	_, hasDesk := raw["deskAlpha"]
	if !hasBar {
		s.BarAlpha = 80
	} else {
		s.BarAlpha = clampAlphaVal(s.BarAlpha)
	}
	if !hasDesk {
		s.DeskAlpha = 70
	} else {
		s.DeskAlpha = clampAlphaVal(s.DeskAlpha)
	}
	if hasBar && hasDesk && s.BarAlpha == 100 && s.DeskAlpha == 100 {
		s.BarAlpha, s.DeskAlpha = 80, 70
		settingsNeedSave = true
	}
	if s.Color == 0 {
		s.Color = 0x00FFFFFF
	}
	if s.DashW < int(dashW) {
		s.DashW = int(dashW)
	}
	if s.DashH < int(dashH+capH) {
		s.DashH = int(dashH + capH)
	}
	return s
}
func saveSettings() {
	_ = os.MkdirAll(appDir(), 0755)
	b, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(filepath.Join(appDir(), "settings.json"), b, 0644)
}
func safeColor() uint32 {
	c := cfg.Color & 0xFFFFFF
	if c == colorKey {
		return 0xFE
	}
	return c
}
func ft64(f filetime) uint64 { return uint64(f.high)<<32 | uint64(f.low) }
func pGB(n, ps uintptr) float64 { return float64(uint64(n)*uint64(ps)) / (1024 * 1024 * 1024) }
func pMB(n, ps uintptr) float64 { return float64(uint64(n)*uint64(ps)) / (1024 * 1024) }

func refreshStats() {
	var idle, kernel, user filetime
	procGetSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	i, k, u := ft64(idle), ft64(kernel), ft64(user)
	if prevKernel != 0 {
		td := (k - prevKernel) + (u - prevUser)
		if td > 0 {
			p := int(((td - (i - prevIdle)) * 100) / td)
			if p < 0 { p = 0 }
			if p > 100 { p = 100 }
			cpuPercent = p
		}
	}
	prevIdle, prevKernel, prevUser = i, k, u
	var mem memoryStatusEx
	mem.length = uint32(unsafe.Sizeof(mem))
	procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	memLoad = int(mem.memoryLoad)
	totalGB = float64(mem.totalPhys) / (1024 * 1024 * 1024)
	usedGB = float64(mem.totalPhys-mem.availPhys) / (1024 * 1024 * 1024)
	var pi perfInfo
	pi.cb = uint32(unsafe.Sizeof(pi))
	procGetPerformanceInfo.Call(uintptr(unsafe.Pointer(&pi)), uintptr(pi.cb))
	ps := pi.pageSize
	if ps == 0 { ps = 4096 }
	var li memListInfo
	procNtQuerySystemInformation.Call(80, uintptr(unsafe.Pointer(&li)), unsafe.Sizeof(li), 0)
	var stby uintptr
	for _, n := range li.priority { stby += n }
	detail.totalGB, detail.usedGB = totalGB, usedGB
	detail.availGB = float64(mem.availPhys) / (1024 * 1024 * 1024)
	detail.usedPct = memLoad
	detail.commitUsedGB, detail.commitLimitGB = pGB(pi.commitTotal, ps), pGB(pi.commitLimit, ps)
	detail.cachedGB = pGB(stby+li.modifiedPage, ps)
	detail.pagedMB, detail.nonpagedMB = pMB(pi.kernelPaged, ps), pMB(pi.kernelNonpaged, ps)
	detail.sysCacheUsedMB, detail.sysCacheTotalMB = pMB(pi.systemCache, ps), pMB(pi.physicalTotal, ps)
}

func hudText() string {
	var p []string
	s := currentSnap()
	if s.HasData && s.Plan != "" {
		p = append(p, s.Plan)
	}
	if cfg.ShowCPU { p = append(p, fmt.Sprintf("CPU  %d%%", cpuPercent)) }
	if cfg.ShowMem { p = append(p, fmt.Sprintf("内存  %d%%   %.1f / %.1f GB", memLoad, usedGB, totalGB)) }
	if cfg.ShowCursor {
		if s.HasData {
			p = append(p, fmt.Sprintf("Cursor 剩余  %d%%", s.RemainingPct))
		} else {
			p = append(p, "Cursor 剩余  —")
		}
	}
	if len(p) == 0 { return "DeskHUD" }
	if len(p) == 1 { return p[0] }
	out := p[0]
	for i := 1; i < len(p); i++ {
		out += "   ·   " + p[i]
	}
	return out
}

func loadPos() (int32, int32, bool) {
	b, err := os.ReadFile(filepath.Join(appDir(), "pos.json"))
	if err != nil { return 0, 0, false }
	var p posFile
	if json.Unmarshal(b, &p) != nil { return 0, 0, false }
	return p.X, p.Y, true
}
func savePos(hwnd uintptr) {
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	_ = os.MkdirAll(appDir(), 0755)
	b, _ := json.Marshal(posFile{X: r.left, Y: r.top})
	_ = os.WriteFile(filepath.Join(appDir(), "pos.json"), b, 0644)
}
func defaultHUDPos() (int32, int32) {
	cx, _, _ := procGetSystemMetrics.Call(smCxScreen)
	if x, y, ok := loadPos(); ok && x < int32(cx) {
		if y < 8 { y = 8 }
		return x, y
	}
	x := int32(cx) - hudW - 20
	if x < 8 { x = 8 }
	return x, 16
}

func mkFont(h int32, weight int, name string) uintptr {
	return mkFontEx(h, weight, name, defaultCharset, clearTypeQual)
}
func mkFontEx(h int32, weight int, name string, charset, quality uint32) uintptr {
	hh := uint32(h)
	if h < 0 { hh = uint32(int32(h)) }
	f, _, _ := procCreateFontW.Call(uintptr(hh), 0, 0, 0, uintptr(weight), 0, 0, 0, uintptr(charset), 0, 0, uintptr(quality), 2, uintptr(unsafe.Pointer(utf16Ptr(name))))
	return f
}
func negH(n int32) uintptr { return uintptr(uint32(n)) } // n is already negative as int32

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
func strokeRound(hdc uintptr, x, y, w, h, r int32, color uint32) {
	nb, _, _ := procGetStockObject.Call(nullBrush)
	pe, _, _ := procCreatePen.Call(psSolid, 1, uintptr(color))
	oldB, _, _ := procSelectObject.Call(hdc, nb)
	oldP, _, _ := procSelectObject.Call(hdc, pe)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h), uintptr(r), uintptr(r))
	procSelectObject.Call(hdc, oldB)
	procSelectObject.Call(hdc, oldP)
	procDeleteObject.Call(pe)
}
func drawCard(hdc uintptr, x, y, w, h, r int32) {
	fillRound(hdc, x, y, w, h, r, colCard)
	strokeRound(hdc, x, y, w, h, r, colLine)
}
func ghostBtn(hdc uintptr, x, y, w, h int32, text string, id int) {
	fillRound(hdc, x, y, w, h, 8, colCard)
	strokeRound(hdc, x, y, w, h, 8, colLine)
	drawStr(hdc, x, y, w, h, text, colWhite, fontBody, dtCenter|dtVCenter)
	addHit(id, x, y, w, h)
}
func primaryBtn(hdc uintptr, x, y, w, h int32, text string, id int) {
	fillRound(hdc, x, y, w, h, 8, colCyan)
	drawStr(hdc, x, y, w, h, text, colInk, fontBody, dtCenter|dtVCenter)
	addHit(id, x, y, w, h)
}
func alphaFromClick(track rect, x int32) int {
	w := track.right - track.left
	if w < 1 {
		return 100
	}
	pct := int((x - track.left) * 100 / w)
	return clampAlphaVal(pct)
}
func drawDot(hdc uintptr, x, y, d int32, color uint32) {
	fillRound(hdc, x, y, d, d, d/2, color)
}
func drawSpark(hdc uintptr, x, y, w, h int32, pct int, col uint32) {
	if w < 8 || h < 6 {
		return
	}
	pe, _, _ := procCreatePen.Call(psSolid, 2, uintptr(col))
	old, _, _ := procSelectObject.Call(hdc, pe)
	n := 14
	amp := 0.25 + 0.7*float64(pct)/100
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		wave := 0.5 + 0.5*math.Sin(t*math.Pi*3+float64(pct)*0.08)
		px := x + int32(t*float64(w))
		py := y + h - int32((0.15+amp*wave)*float64(h))
		if i == 0 {
			procMoveToEx.Call(hdc, uintptr(px), uintptr(py), 0)
		} else {
			procLineTo.Call(hdc, uintptr(px), uintptr(py))
		}
	}
	procSelectObject.Call(hdc, old)
	procDeleteObject.Call(pe)
}
func drawSlider(hdc uintptr, x, y, w int32, pct int, label string, id int) rect {
	if pct < 30 {
		pct = 30
	}
	if pct > 100 {
		pct = 100
	}
	drawStr(hdc, x, y, 110, 22, label, colWhite, fontBody, dtLeft|dtVCenter)
	tx, ty, tw, th := x+120, y+6, w-170, int32(10)
	fillRound(hdc, tx, ty, tw, th, th/2, colTrack)
	fw := tw * int32(pct) / 100
	if fw < th {
		fw = th
	}
	fillRound(hdc, tx, ty, fw, th, th/2, colCyan)
	kx := tx + fw - 7
	if kx < tx {
		kx = tx
	}
	fillRound(hdc, kx, ty-3, 16, 16, 8, colWhite)
	drawStr(hdc, x+w-42, y, 42, 22, fmt.Sprintf("%d%%", pct), colMute, fontSmall, dtRight|dtVCenter)
	tr := rect{tx, y, tx + tw, y + 22}
	addHit(id, tr.left, tr.top, tr.right-tr.left, tr.bottom-tr.top)
	return tr
}
func fillRectC(hdc uintptr, x, y, w, h int32, color uint32) {
	ob, op, br, pe := selBrush(hdc, color)
	procRectangle.Call(hdc, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h))
	unsel(hdc, ob, op, br, pe)
}
func drawStr(hdc uintptr, x, y, w, h int32, s string, color uint32, fnt uintptr, flags uint32) {
	if fnt != 0 { procSelectObject.Call(hdc, fnt) }
	procSetBkMode.Call(hdc, transparent)
	procSetTextColor.Call(hdc, uintptr(color))
	u, _ := syscall.UTF16FromString(s)
	rc := rect{x, y, x + w, y + h}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&rc)), uintptr(flags|dtSingle))
}
func addHit(id int, x, y, w, h int32) {
	y += hitOff
	hits = append(hits, hit{id, rect{x, y, x + w, y + h}})
}
func gapX() int32 {
	e := uiW - 712
	if e < 0 {
		return 0
	}
	return e
}
func bar(hdc uintptr, x, y, w, h int32, pct int, fill uint32) {
	if pct < 0 { pct = 0 }
	if pct > 100 { pct = 100 }
	fillRound(hdc, x, y, w, h, h, colTrack)
	fw := w * int32(pct) / 100
	if pct > 0 && fw < 2 {
		fw = 2
	}
	if pct > 0 {
		fillRound(hdc, x, y, fw, h, h, fill)
	}
}

func isUltraPlan(name string) bool {
	return strings.Contains(strings.ToLower(name), "ultra")
}

func pill(hdc uintptr, x, y, w, h int32, text string, bg, fg uint32) {
	fillRound(hdc, x, y, w, h, h/2, bg)
	drawStr(hdc, x, y, w, h, text, fg, fontSmall, dtCenter|dtVCenter)
}

func paintCaption(hdc uintptr, cw int32) {
	fillRectC(hdc, 0, 0, cw, capH, colCard)
	if ico := loadAppIcon(); ico != 0 {
		procDrawIconEx.Call(hdc, 10, 6, ico, 24, 24, 0, 0, diNormal)
	}
	drawStr(hdc, 40, 0, 360, capH, appTitle, colWhite, fontBody, dtLeft|dtVCenter)
	bx := cw - 108
	ghostBtn(hdc, bx, 6, 32, 24, "—", hitCapMin)
	maxLab := "□"
	if z, _, _ := procIsZoomed.Call(hwndDash); z != 0 {
		maxLab = "❐"
	}
	ghostBtn(hdc, bx+36, 6, 32, 24, maxLab, hitCapMax)
	fillRound(hdc, bx+72, 6, 32, 24, 8, 0x00303080)
	drawStr(hdc, bx+72, 6, 32, 24, "×", colWhite, fontBody, dtCenter|dtVCenter)
	addHit(hitCapClose, bx+72, 6, 32, 24)
}

func paintDash(hwnd uintptr) {
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 { return }
	var rc rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	cw, ch := rc.right, rc.bottom
	if cw < 1 || ch < 1 {
		procReleaseDC.Call(hwnd, hdc)
		return
	}
	mem, _, _ := procCreateCompatibleDC.Call(hdc)
	if mem == 0 {
		procReleaseDC.Call(hwnd, hdc)
		return
	}
	bmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(cw), uintptr(ch))
	if bmp == 0 {
		procDeleteDC.Call(mem)
		procReleaseDC.Call(hwnd, hdc)
		return
	}
	old, _, _ := procSelectObject.Call(mem, bmp)
	procFillRect.Call(mem, uintptr(unsafe.Pointer(&rc)), boardBrush)
	hits = hits[:0]
	hitOff = 0
	uiW = cw - 48
	if uiW < 712 {
		uiW = 712
	}
	paintCaption(mem, cw)
	hitOff = capH
	procSetViewportOrgEx.Call(mem, 0, capH, 0)

	s := currentSnap()
	if ico := loadAppIcon(); ico != 0 {
		procDrawIconEx.Call(mem, 24, 14, ico, 36, 36, 0, 0, diNormal)
	}
	drawStr(mem, 70, 12, 320, 28, appTitle, colWhite, fontTitle, dtLeft|dtVCenter)
	sub := "Cookie 仅保存在本机"
	if s.Status != "" {
		sub = s.Status + " · Cookie 仅保存在本机"
	}
	drawStr(mem, 70, 38, 360, 18, sub, colMute, fontSmall, dtLeft|dtVCenter)
	gx := gapX()
	pill(mem, 430+gx, 18, 58, 22, "v"+appVersion, colInner, colMute)
	ghostBtn(mem, 496+gx, 14, 68, 28, "Cookie", hitCookie)
	ghostBtn(mem, 572+gx, 14, 60, 28, "刷新", hitRefresh)

	px, py := int32(640)+gx, int32(16)
	fillRound(mem, px, py, 44, 24, 12, map[bool]uint32{true: colCyan, false: colTrack}[cfg.Pinned])
	knob := px + 4
	if cfg.Pinned { knob = px + 22 }
	fillRound(mem, knob, py+3, 18, 18, 9, colWhite)
	drawStr(mem, 688+gx, 16, 48, 24, map[bool]string{true: "置顶", false: "普通"}[cfg.Pinned], colMute, fontSmall, dtLeft|dtVCenter)
	addHit(hitPin, px, py, 100, 24)

	tabs := []string{"概览", "模型", "系统", "设置"}
	tx, ty, tw, th := int32(24), int32(72), int32(88), int32(32)
	fillRound(mem, tx, ty, tw*4, th, 16, colInner)
	strokeRound(mem, tx, ty, tw*4, th, 16, colLine)
	ids := []int{hitTab0, hitTab1, hitTab2, hitTab3}
	for i, name := range tabs {
		x := tx + int32(i)*tw
		fg := uint32(colMute)
		if page == i {
			fillRound(mem, x+3, ty+3, tw-6, th-6, 13, colCyan)
			fg = colInk
		}
		drawStr(mem, x, ty, tw, th, name, fg, fontBody, dtCenter|dtVCenter)
		addHit(ids[i], x, ty, tw, th)
	}

	switch page {
	case 0:
		paintOverview(mem)
	case 1:
		paintModels(mem)
	case 2:
		paintSystem(mem)
	default:
		paintSettings(mem)
	}
	fy := rc.bottom - capH - 22
	if fy < 662 {
		fy = 662
	}
	drawStr(mem, 24, fy, uiW, 20, "关主窗口后看右下角托盘 · 用量约 3 分钟刷新一次", colMute, fontSmall, dtLeft|dtVCenter)
	procSetViewportOrgEx.Call(mem, 0, 0, 0)
	hitOff = 0
	const srcCopy = 0x00CC0020
	procBitBlt.Call(hdc, 0, 0, uintptr(cw), uintptr(ch), mem, 0, 0, srcCopy)
	procSelectObject.Call(mem, old)
	procDeleteObject.Call(bmp)
	procDeleteDC.Call(mem)
	procReleaseDC.Call(hwnd, hdc)
}

func paintOverview(hdc uintptr) {
	s := currentSnap()
	remain, plan, foot := "—", "未连接", "设置里粘贴 Cookie 后显示剩余额度"
	rp, prem := 0, 0
	if s.HasData {
		remain = fmt.Sprintf("%d%%", s.RemainingPct)
		plan = s.Plan
		foot = "Cursor 模型剩余"
		if s.CycleEnd != "" {
			foot += "  ·  至 " + s.CycleEnd
		}
		rp = s.RemainingPct
		if s.OtherPct > 0 {
			prem = 100 - s.OtherPct
			if prem < 0 {
				prem = 0
			}
		}
	}
	premTxt := "—"
	if s.HasData && s.OtherPct > 0 {
		premTxt = fmt.Sprintf("%d%%", prem)
	}

	gx := gapX()
	leftW := 492 + gx
	rx := 528 + gx
	drawCard(hdc, 24, 112, leftW, 158, 16)
	drawStr(hdc, 40, 122, 200, 58, remain, colCyan, fontHuge, dtLeft|dtVCenter)
	pill(hdc, 40, 184, 72, 22, plan, colCyan, colInk)
	drawStr(hdc, 40, 212, 220, 18, foot, colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, 248, 132, 140, 18, "Cursor 剩余", colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, 400+gx, 132, 96, 18, remain, colCyan, fontSmall, dtRight|dtVCenter)
	bar(hdc, 248, 152, 248+gx, 8, rp, colCyan)
	drawStr(hdc, 248, 176, 140, 18, "高级剩余", colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, 400+gx, 176, 96, 18, premTxt, colLilac, fontSmall, dtRight|dtVCenter)
	bar(hdc, 248, 196, 248+gx, 8, prem, colLilac)
	drawStr(hdc, 248, 218, 248+gx, 18, formatWan(s.TotalTokens)+" token", colMute, fontSmall, dtLeft|dtVCenter)

	drawCard(hdc, rx, 112, 208, 74, 14)
	drawStr(hdc, rx+12, 118, 70, 18, "CPU", colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, rx+72, 116, 120, 28, fmt.Sprintf("%d%%", cpuPercent), colWhite, fontTitle, dtLeft|dtVCenter)
	drawSpark(hdc, rx+12, 142, 184, 34, cpuPercent, colCyan)

	drawCard(hdc, rx, 196, 208, 74, 14)
	drawStr(hdc, rx+12, 202, 70, 18, "内存", colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, rx+72, 200, 120, 28, fmt.Sprintf("%d%%", memLoad), colWhite, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, rx+12, 224, 184, 18, fmt.Sprintf("%.1f / %.1f GB", usedGB, totalGB), colMute, fontSmall, dtLeft|dtVCenter)
	drawSpark(hdc, rx+12, 240, 184, 22, memLoad, colLilac)

	cursorVal, cursorSub, cursorUsed := "—", "本月消耗", "—"
	otherVal, otherSub, otherUsed := "—", "套餐额度", "—"
	if s.HasData {
		cursorVal = formatEstimate(s.CursorUSD, s.HasCursorUSD)
		if s.CursorPct > 0 {
			cursorUsed = fmt.Sprintf("已用 %d%%", s.CursorPct)
		}
		otherVal = formatOver(s.UsedUSD, s.LimitUSD)
		if s.HasOtherUSD {
			otherSub = formatEstimate(s.OtherUSD, true)
		}
		if s.OverUSD > 0 {
			otherSub = "超出会员 " + formatMoney(s.OverUSD)
		}
		if s.OtherPct > 0 {
			otherUsed = fmt.Sprintf("已用 %d%%", s.OtherPct)
		}
	}
	half := (uiW - 12) / 2
	drawCard(hdc, 24, 282, half, 108, 16)
	drawStr(hdc, 40, 292, 220, 18, "Cursor 模型消耗", colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, 40, 312, half-32, 36, cursorVal, colCyan, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, 40, 350, 160, 18, cursorSub, colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, 24+half-166, 350, 150, 18, cursorUsed, colCyan, fontSmall, dtRight|dtVCenter)
	bar(hdc, 40, 370, half-32, 10, s.CursorPct, colCyan)

	ox := 24 + half + 12
	drawCard(hdc, ox, 282, uiW-half-12, 108, 16)
	drawStr(hdc, ox+16, 292, 220, 18, "高级模型消耗", colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, ox+16, 312, half-32, 36, otherVal, colLilac, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, ox+16, 350, 160, 18, otherSub, colMute, fontSmall, dtLeft|dtVCenter)
	drawStr(hdc, ox+half-166, 350, 150, 18, otherUsed, colLilac, fontSmall, dtRight|dtVCenter)
	bar(hdc, ox+16, 370, half-32, 10, s.OtherPct, colLilac)

	drawCard(hdc, 24, 402, uiW, 112, 16)
	drawStr(hdc, 40, 410, 200, 28, "今日 token", colWhite, fontTitle, dtLeft|dtVCenter)
	today := [][2]string{{"输入", "—"}, {"输出", "—"}, {"缓存读", "—"}, {"缓存写", "—"}}
	if s.HasData && s.HasToday {
		today = [][2]string{
			{"输入", formatWan(s.TodayIn)},
			{"输出", formatWan(s.TodayOut)},
			{"缓存读", formatWan(s.TodayCacheR)},
			{"缓存写", formatWan(s.TodayCacheW)},
		}
	} else if s.HasData {
		today = [][2]string{
			{"输入", formatWan(s.InputTokens)},
			{"输出", formatWan(s.OutputTokens)},
			{"缓存读", "—"},
			{"缓存写", "—"},
		}
	}
	cellW := (uiW - 48) / 4
	if cellW < 160 {
		cellW = 160
	}
	for i, m := range today {
		x := int32(40) + int32(i)*(cellW+12)
		fillRound(hdc, x, 438, cellW, 62, 12, colInner)
		drawStr(hdc, x+12, 444, cellW-24, 18, m[0], colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, x+12, 464, cellW-24, 28, m[1], colCyan, fontTitle, dtLeft|dtVCenter)
	}

	drawCard(hdc, 24, 526, uiW, 88, 16)
	drawStr(hdc, 40, 534, 160, 28, "Grok 机器人", colWhite, fontTitle, dtLeft|dtVCenter)
	if s.HasData && isUltraPlan(s.Plan) {
		used := "已用 —"
		if s.GrokPct > 0 {
			used = fmt.Sprintf("已用 %d%%", s.GrokPct)
		}
		reset := "刷新时间未返回"
		if s.GrokReset != "" {
			reset = s.GrokReset
		}
		drawStr(hdc, 200, 536, 120, 22, used, colOrange, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, 320, 536, uiW-320, 22, reset, colMute, fontSmall, dtLeft|dtVCenter)
		bar(hdc, 40, 566, uiW-32, 12, s.GrokPct, colOrange)
		drawStr(hdc, 40, 586, 400, 16, "周额度，到点刷新", colMute, fontSmall, dtLeft)
	} else {
		drawStr(hdc, 40, 564, uiW-48, 28, "升级 Ultra 后可查看 Grok 机器人用量", colMute, fontBody, dtLeft|dtVCenter)
	}
}

func paintModels(hdc uintptr) {
	s := currentSnap()
	half := (uiW - 12) / 2
	ox := 24 + half + 12
	drawCard(hdc, 24, 112, half, 528, 20)
	drawCard(hdc, ox, 112, uiW-half-12, 528, 20)
	if !s.HasData || len(s.Models) == 0 {
		msg := "粘贴 Cookie 并刷新后，这里显示各模型消耗"
		if s.HasData {
			msg = "这个账期还没有按模型拆开的 token"
		}
		drawStr(hdc, 44, 140, half-40, 40, msg, colMute, fontBody, dtLeft)
		drawStr(hdc, ox+20, 140, half-40, 40, msg, colMute, fontBody, dtLeft)
		return
	}
	var cursor, other []ModelRow
	for _, m := range s.Models {
		if m.Group == "cursor" {
			cursor = append(cursor, m)
		} else {
			other = append(other, m)
		}
	}
	drawGroup := func(x int32, title string, rows []ModelRow, tint uint32) {
		drawDot(hdc, x+20, 130, 10, tint)
		drawStr(hdc, x+38, 122, 200, 28, title, colWhite, fontTitle, dtLeft|dtVCenter)
		sumT, sumU, hasU := 0.0, 0.0, false
		for _, m := range rows {
			sumT += m.Tokens
			if m.HasPrice {
				sumU += m.EstUSD
				hasU = true
			} else if m.Amount > 0 {
				sumU += m.Amount
				hasU = true
			}
		}
		drawStr(hdc, x+20, 154, 80, 18, "总用量", colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, x+20, 172, half-40, 28, formatWan(sumT), tint, fontTitle, dtLeft|dtVCenter)
		drawStr(hdc, x+20, 204, 80, 18, "总金额", colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, x+20, 222, half-40, 28, formatEstimate(sumU, hasU), tint, fontTitle, dtLeft|dtVCenter)
		if len(rows) == 0 {
			drawStr(hdc, x+20, 262, half-40, 20, "无", colMute, fontSmall, dtLeft)
			return
		}
		max := 1.0
		for _, m := range rows {
			w := m.Tokens
			if w <= 0 {
				w = m.Amount
			}
			if w > max {
				max = w
			}
		}
		n := len(rows)
		if n > 8 {
			n = 8
		}
		y := int32(262)
		for i := 0; i < n; i++ {
			m := rows[i]
			val := formatWan(m.Tokens)
			weight := m.Tokens
			if m.Tokens <= 0 && m.Amount > 0 {
				val = formatMoney(m.Amount)
				weight = m.Amount
			}
			drawStr(hdc, x+20, y, half-148, 18, m.Name, colWhite, fontSmall, dtLeft|dtVCenter)
			drawStr(hdc, x+half-128, y, 108, 18, val, colMute, fontSmall, dtRight|dtVCenter)
			bar(hdc, x+20, y+20, half-40, 8, int(weight*100/max), tint)
			y += 44
		}
	}
	drawGroup(24, "Cursor 模型", cursor, colCyan)
	drawGroup(ox, "高级模型", other, colLilac)
}

func paintSystem(hdc uintptr) {
	drawCard(hdc, 24, 112, uiW, 528, 20)
	drawStr(hdc, 44, 130, 120, 28, "内存", colWhite, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, 44, 160, 300, 22, fmt.Sprintf("%.1f/%.1f GB (%d%%)", detail.usedGB, detail.totalGB, detail.usedPct), colMute, fontBody, dtLeft)
	bar(hdc, 44, 190, uiW-56, 10, detail.usedPct, colGreen)
	rows := [][4]string{
		{"使用中", fmt.Sprintf("%.1f GB", detail.usedGB), "可用", fmt.Sprintf("%.1f GB", detail.availGB)},
		{"已提交", fmt.Sprintf("%.1f/%.1f GB", detail.commitUsedGB, detail.commitLimitGB), "已缓存", fmt.Sprintf("%.1f GB", detail.cachedGB)},
		{"分页缓冲池", fmt.Sprintf("%.0f MB", detail.pagedMB), "非分页缓冲池", fmt.Sprintf("%.0f MB", detail.nonpagedMB)},
	}
	innerW := (uiW - 56) / 2
	rx := 44 + innerW + 12
	y := int32(220)
	for _, r := range rows {
		fillRound(hdc, 44, y, innerW, 56, 12, colInner)
		fillRound(hdc, rx, y, uiW-(rx-24)-20, 56, 12, colInner)
		drawStr(hdc, 56, y+8, 160, 18, r[0], colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, 56, y+26, innerW-24, 22, r[1], colWhite, fontTitle, dtLeft|dtVCenter)
		drawStr(hdc, rx+12, y+8, 160, 18, r[2], colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(hdc, rx+12, y+26, innerW-24, 22, r[3], colWhite, fontTitle, dtLeft|dtVCenter)
		y += 68
	}
	drawStr(hdc, 44, 438, 160, 28, "系统缓存", colWhite, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, 44, 466, 400, 22, fmt.Sprintf("%.2f / %.2f MB", detail.sysCacheUsedMB, detail.sysCacheTotalMB), colMute, fontBody, dtLeft|dtVCenter)
	cp := 0
	if detail.sysCacheTotalMB > 0 {
		cp = int(detail.sysCacheUsedMB * 100 / detail.sysCacheTotalMB)
	}
	bar(hdc, 44, 500, uiW-56, 8, cp, colCyan)
}

func paintSettings(hdc uintptr) {
	drawCard(hdc, 24, 112, uiW, 308, 16)
	drawStr(hdc, 44, 122, 280, 28, "桌面显示", colWhite, fontTitle, dtLeft|dtVCenter)
	drawStr(hdc, 44, 148, 300, 18, "只影响顶部栏和桌面监控面板", colMute, fontSmall, dtLeft|dtVCenter)

	box := func(x, y int32, on bool, id int, label string) {
		bg := uint32(colTrack)
		if on {
			bg = colCyan
		}
		fillRound(hdc, x, y, 20, 20, 6, bg)
		if on {
			drawStr(hdc, x, y, 20, 20, "✓", colInk, fontSmall, dtCenter|dtVCenter)
		}
		drawStr(hdc, x+28, y, 220, 20, label, colWhite, fontBody, dtLeft|dtVCenter)
		addHit(id, x, y, 248, 20)
	}
	box(44, 176, cfg.ShowCPU, hitCpu, "显示 CPU")
	box(44, 208, cfg.ShowMem, hitMem, "显示内存")
	box(44, 240, cfg.ShowCursor, hitCursor, "显示模型剩余")
	box(44, 272, cfg.ShowBar, hitBar, "顶部显示栏")
	box(44, 304, cfg.ShowDesk, hitDesk, "桌面监控面板")

	c := safeColor()
	drawStr(hdc, 360, 176, 80, 28, "文字颜色", colWhite, fontBody, dtLeft|dtVCenter)
	fillRound(hdc, 448, 178, 28, 28, 8, c)
	strokeRound(hdc, 448, 178, 28, 28, 8, colLine)
	ghostBtn(hdc, 488, 178, 72, 28, "选择", hitColor)
	drawStr(hdc, 568, 178, 140, 28, fmt.Sprintf("#%02X%02X%02X", c&0xFF, (c>>8)&0xFF, (c>>16)&0xFF), colMute, fontSmall, dtLeft|dtVCenter)

	sw := 352 + gapX()
	trackBarAlpha = drawSlider(hdc, 360, 228, sw, clampAlphaVal(cfg.BarAlpha), "顶栏透明度", hitBarAlpha)
	trackDeskAlpha = drawSlider(hdc, 360, 276, sw, clampAlphaVal(cfg.DeskAlpha), "面板透明度", hitDeskAlpha)

	drawCard(hdc, 24, 432, uiW, 200, 16)
	drawStr(hdc, 44, 442, 200, 28, "账户与更新", colWhite, fontTitle, dtLeft|dtVCenter)
	login := "未登录。点右上角 Cookie，粘贴 WorkosCursorSessionToken。"
	if hasCookie() {
		login = "已保存登录态（只在本机）"
	}
	drawStr(hdc, 44, 470, 660, 20, login, colMute, fontSmall, dtLeft|dtVCenter)

	primaryBtn(hdc, 44, 504, 108, 32, "检查更新", hitCheckUpdate)
	ghostBtn(hdc, 164, 504, 80, 32, "退出", hitQuit)
	ghostBtn(hdc, 256, 504, 80, 32, "卸载", hitUninstall)
	drawStr(hdc, 352, 508, 120, 24, "v"+appVersion, colMute, fontSmall, dtLeft|dtVCenter)
	if cfg.SkipVersion != "" {
		drawStr(hdc, 440, 508, 260, 24, "已忽略  v"+cfg.SkipVersion, colMute, fontSmall, dtLeft|dtVCenter)
	}
	drawStr(hdc, 44, 552, 660, 18, "透明度只作用于顶栏和面板。", colMute, fontSmall, dtLeft)
	drawStr(hdc, 44, 576, 660, 18, "卸载走正常向导，会清干净。", colMute, fontSmall, dtLeft)
}

func hitTest(x, y int32) int {
	for _, h := range hits {
		if x >= h.r.left && x < h.r.right && y >= h.r.top && y < h.r.bottom {
			return h.id
		}
	}
	return 0
}

func applyClick(id int) {
	switch id {
	case hitTab0:
		page = 0
		pendingDash = true
	case hitTab1:
		page = 1
		pendingDash = true
	case hitTab2:
		page = 2
		pendingDash = true
	case hitTab3:
		page = 3
		pendingDash = true
	case hitCookie:
		pendingCookie = true
	case hitRefresh:
		if hasCookie() {
			snapMu.Lock()
			snap.Status = "正在刷新…"
			snapMu.Unlock()
			refreshCursorAsync()
			pendingDash = true
		} else {
			pendingCookie = true
		}
	case hitCursor:
		cfg.ShowCursor = !cfg.ShowCursor
		saveSettings()
		pendingHUD, pendingDash, pendingDesk = true, true, true
	case hitBar:
		cfg.ShowBar = !cfg.ShowBar
		saveSettings()
		applyOverlayVis()
		applyHUDAlpha()
		pendingDash = true
	case hitDesk:
		cfg.ShowDesk = !cfg.ShowDesk
		saveSettings()
		applyOverlayVis()
		applyDeskAlpha()
		pendingDash = true
	case hitBarAlpha:
		cfg.BarAlpha = alphaFromClick(trackBarAlpha, clickX)
		saveSettings()
		lastHUDAlpha = 0
		applyHUDAlpha()
		pendingDash = true
	case hitDeskAlpha:
		cfg.DeskAlpha = alphaFromClick(trackDeskAlpha, clickX)
		saveSettings()
		lastDeskAlpha = 0
		applyDeskAlpha()
		pendingDash, pendingDesk = true, true
	case hitPin:
		cfg.Pinned = !cfg.Pinned
		saveSettings()
		applyPin()
		pendingDash = true
	case hitCpu:
		cfg.ShowCPU = !cfg.ShowCPU
		saveSettings()
		pendingHUD, pendingDash, pendingDesk = true, true, true
	case hitMem:
		cfg.ShowMem = !cfg.ShowMem
		saveSettings()
		pendingHUD, pendingDash, pendingDesk = true, true, true
	case hitColor:
		pendingColor = true
	case hitQuit:
		if hwndHUD != 0 {
			procDestroyWindow.Call(hwndHUD)
		}
	case hitUninstall:
		pendingUninstall = true
	case hitCheckUpdate:
		go func() {
			checkUpdateNow()
		}()
	case hitCapMin:
		if hwndDash != 0 {
			procShowWindow.Call(hwndDash, swMinimize)
		}
	case hitCapMax:
		if hwndDash == 0 {
			return
		}
		if z, _, _ := procIsZoomed.Call(hwndDash); z != 0 {
			procShowWindow.Call(hwndDash, swRestore)
		} else {
			procShowWindow.Call(hwndDash, swMaximize)
		}
		pendingDash = true
	case hitCapClose:
		if hwndDash != 0 {
			procShowWindow.Call(hwndDash, swHide)
			setHUDClickThrough(true)
		}
	}
}

func applyPin() {
	if hwndDash == 0 { return }
	top := hwndNoTopmost
	if cfg.Pinned { top = hwndTopmost }
	procSetWindowPos.Call(hwndDash, top, 0, 0, 0, 0, swpNoSize|swpNoActivate|0x0002) // NOSIZE|NOMOVE|NOACTIVATE
}

func pickColor() {
	cc := chooseColorW{lStructSize: uint32(unsafe.Sizeof(chooseColorW{})), hwndOwner: hwndDash, rgbResult: safeColor(), lpCustColors: uintptr(unsafe.Pointer(&custColors[0])), flags: ccRGBInit | ccFullOpen | ccAnyColor}
	ok, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if ok != 0 {
		cfg.Color = cc.rgbResult
		saveSettings()
		pendingHUD, pendingDash, pendingDesk = true, true, true
	}
}


func readClipboard() string {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer procCloseClipboard.Call()
	h, _, _ := procGetClipboardData.Call(13) // CF_UNICODETEXT
	if h == 0 {
		return ""
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(p))[:])
}

func applyClipboardCookie() {
	text := readClipboard()
	if strings.TrimSpace(text) == "" {
		alert(appTitle, "请先复制 Cookie，再点 Cookie。可复制整段，或只复制 WorkosCursorSessionToken 的值。")
		return
	}
	if err := saveCookie(text); err != nil {
		alert(appTitle, err.Error())
		return
	}
	snapMu.Lock()
	snap.Status = "已保存 Cookie，正在读取用量…"
	snapMu.Unlock()
	refreshCursorAsync()
	pendingDash = true
}

func addTray(hwnd uintptr) {
	if trayAdded || hwnd == 0 {
		return
	}
	ico := loadAppIcon()
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hwnd = hwnd
	nid.uID = 1
	nid.uFlags = nifMessage | nifIcon | nifTip
	nid.uCallbackMessage = wmTray
	nid.hIcon = ico
	u, _ := syscall.UTF16FromString(appTitle)
	copy(nid.szTip[:], u)
	r, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	trayAdded = r != 0
}

func delTray(hwnd uintptr) {
	if !trayAdded || hwnd == 0 {
		return
	}
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hwnd = hwnd
	nid.uID = 1
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	trayAdded = false
}

func showTrayMenu() {
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	menu, _, _ := procCreatePopupMenu.Call()
	procAppendMenuW.Call(menu, mfString, idOpen, uintptr(unsafe.Pointer(utf16Ptr("打开界面"))))
	procAppendMenuW.Call(menu, mfString, idQuit, uintptr(unsafe.Pointer(utf16Ptr("退出"))))
	if hwndHUD != 0 {
		procSetForegroundWindow.Call(hwndHUD)
	}
	procTrackPopupMenu.Call(menu, tpmRightbutton|tpmBottomalign|tpmRightalign, uintptr(pt.x), uintptr(pt.y), 0, hwndHUD, 0)
	procDestroyMenu.Call(menu)
	if hwndHUD != 0 {
		procPostMessageW.Call(hwndHUD, wmNull, 0, 0)
	}
}

func saveDashSize(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	if z, _, _ := procIsZoomed.Call(hwnd); z != 0 {
		return
	}
	if ic, _, _ := procIsIconic.Call(hwnd); ic != 0 {
		return
	}
	var wr rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	w, h := int(wr.right-wr.left), int(wr.bottom-wr.top)
	if w < int(dashW) {
		w = int(dashW)
	}
	if h < int(dashH+capH) {
		h = int(dashH + capH)
	}
	if cfg.DashW == w && cfg.DashH == h {
		return
	}
	cfg.DashW, cfg.DashH = w, h
	saveSettings()
}

func openDash() {
	if hwndDash != 0 {
		if ic, _, _ := procIsIconic.Call(hwndDash); ic != 0 {
			procShowWindow.Call(hwndDash, swRestore)
		} else {
			procShowWindow.Call(hwndDash, swShow)
		}
		applyPin()
		setHUDClickThrough(false)
		pendingDash = true
		return
	}
	dw, dh := int32(cfg.DashW), int32(cfg.DashH)
	if dw < dashW {
		dw = dashW
	}
	if dh < dashH+capH {
		dh = dashH + capH
	}
	cx, _, _ := procGetSystemMetrics.Call(smCxScreen)
	cy, _, _ := procGetSystemMetrics.Call(smCyScreen)
	x := (int32(cx) - dw) / 2
	y := (int32(cy) - dh) / 2
	style := wsPopup | wsThickframe | wsVisible | wsClipChildren
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(utf16Ptr("DeskHUDDash"))), uintptr(unsafe.Pointer(utf16Ptr(appTitle))),
		uintptr(style), uintptr(x), uintptr(y), uintptr(dw), uintptr(dh), 0, 0, hInst, 0)
	if hwnd == 0 {
		alert(appTitle, "窗口创建失败")
		return
	}
	hwndDash = hwnd
	applyDashChrome(hwnd)
	applyPin()
	setHUDClickThrough(false)
	pendingDash = true
}

func applyDashChrome(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	c := uint32(colBoard)
	procDwmSetWindowAttribute.Call(hwnd, dwmBorderColor, uintptr(unsafe.Pointer(&c)), 4)
	procDwmSetWindowAttribute.Call(hwnd, dwmCaptionColor, uintptr(unsafe.Pointer(&c)), 4)
}

func dashNCHitTest(hwnd, lParam uintptr) uintptr {
	x := int32(int16(lParam & 0xFFFF))
	y := int32(int16((lParam >> 16) & 0xFFFF))
	var wr rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	lx, ly := x-wr.left, y-wr.top
	w, h := wr.right-wr.left, wr.bottom-wr.top
	b := int32(frameHit)
	left, right, top, bottom := lx < b, lx >= w-b, ly < b, ly >= h-b
	switch {
	case top && left:
		return htTopLeft
	case top && right:
		return htTopRight
	case bottom && left:
		return htBottomLeft
	case bottom && right:
		return htBottomRight
	case left:
		return htLeft
	case right:
		return htRight
	case top:
		return htTop
	case bottom:
		return htBottom
	}
	if ly >= 0 && ly < capH {
		id := hitTest(lx, ly)
		if id == hitCapMin || id == hitCapMax || id == hitCapClose {
			return htClient
		}
		return htCaption
	}
	return htClient
}

func hudProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		paintHUDTo(hwnd, hdc, true)
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmAppRefresh:
		pendingHUD, pendingDash, pendingDesk = true, true, true
		return 0
	case wmTray:
		switch lParam {
		case wmRButtonUp, wmContextMenu:
			pendingTray = true
		case wmLButtonUp, wmLButtonDblClk:
			pendingOpen = true
		}
		return 0
	case wmSetCursor:
		if arrowCursor != 0 { procSetCursor.Call(arrowCursor) }
		return 1
	case wmLButtonDown:
		var pt point
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		var r rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		dragOffX, dragOffY = pt.x-r.left, pt.y-r.top
		dragging = true
		dragHWND = hwnd
		procSetCapture.Call(hwnd)
		return 0
	case wmMouseMove:
		if dragging {
			var pt point
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			moveX, moveY = pt.x-dragOffX, pt.y-dragOffY
			pendingMove = true
		}
		return 0
	case wmLButtonUp:
		if dragging {
			dragging = false
			dragHWND = 0
			procReleaseCapture.Call()
			savePos(hwnd)
		}
		return 0
	case wmRButtonUp:
		var pt point
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		menu, _, _ := procCreatePopupMenu.Call()
		procAppendMenuW.Call(menu, mfString, idOpen, uintptr(unsafe.Pointer(utf16Ptr("打开界面"))))
		procAppendMenuW.Call(menu, mfString, idQuit, uintptr(unsafe.Pointer(utf16Ptr("退出"))))
		procTrackPopupMenu.Call(menu, tpmRightbutton, uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
		procDestroyMenu.Call(menu)
		return 0
	case wmCommand:
		switch wParam & 0xffff {
		case idOpen:
			pendingOpen = true
		case idQuit:
			procDestroyWindow.Call(hwnd)
		}
		return 0
	case wmDestroy:
		savePos(hwnd)
		delTray(hwnd)
		if hwndDash != 0 { procDestroyWindow.Call(hwndDash) }
		if hwndDesk != 0 { procDestroyWindow.Call(hwndDesk) }
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func dashProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmPaint:
		var ps paintStruct
		procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		pendingDash = true
		return 0
	case wmNCCalcSize:
		return 0
	case wmNCHitTest:
		return dashNCHitTest(hwnd, lParam)
	case wmGetMinMaxInfo:
		r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
		if lParam != 0 {
			mm := (*minmaxinfo)(unsafe.Pointer(lParam))
			mm.minTrack.x = dashW
			mm.minTrack.y = dashH + capH
		}
		return r
	case wmSize:
		if wParam != sizeMinimized {
			saveDashSize(hwnd)
			pendingDash = true
		}
		return 0
	case wmLButtonDown:
		clickX = int32(int16(lParam & 0xFFFF))
		clickY = int32(int16((lParam >> 16) & 0xFFFF))
		id := hitTest(clickX, clickY)
		if id == hitBarAlpha || id == hitDeskAlpha {
			slideID = id
			procSetCapture.Call(hwnd)
			pendingSlide = true
			return 0
		}
		if clickY < capH {
			if id == hitCapMin || id == hitCapMax || id == hitCapClose {
				pendingClick = true
				return 0
			}
			procReleaseCapture.Call()
			procSendMessageW.Call(hwnd, wmNCLButtonDown, htCaption, 0)
			return 0
		}
		pendingClick = true
		return 0
	case wmMouseMove:
		if slideID != 0 {
			clickX = int32(int16(lParam & 0xFFFF))
			pendingSlide = true
			return 0
		}
	case wmLButtonUp:
		if slideID != 0 {
			clickX = int32(int16(lParam & 0xFFFF))
			pendingSlide = true
			pendingSlideEnd = true
			procReleaseCapture.Call()
			return 0
		}
	case wmClose:
		procShowWindow.Call(hwnd, swHide)
		setHUDClickThrough(true)
		return 0
	case wmDestroy:
		hwndDash = 0
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func registerClass(name string, cb uintptr, bg syscall.Handle) bool {
	ico := loadAppIcon()
	wc := wndClassEx{size: uint32(unsafe.Sizeof(wndClassEx{})), wndProc: cb, instance: syscall.Handle(hInst), icon: syscall.Handle(ico), cursor: syscall.Handle(arrowCursor), background: bg, className: utf16Ptr(name), iconSm: syscall.Handle(ico)}
	a, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return a != 0
}

func main() {
	runtime.LockOSThread()
	defer func() {
		if r := recover(); r != nil { alert(appTitle, fmt.Sprintf("启动失败: %v", r)) }
	}()
	if len(os.Args) > 1 && os.Args[1] == "--unregister" {
		unregisterApp()
		return
	}
	procSetProcessDPIAware.Call()
	cfg = loadSettings()
	if settingsNeedSave {
		saveSettings()
	}
	custColors[0] = 0x00FFFFFF
	refreshStats()
	time.Sleep(200 * time.Millisecond)
	refreshStats()

	hInst, _, _ = procGetModuleHandleW.Call(0)
	arrowCursor, _, _ = procLoadCursorW.Call(0, idcArrow)
	keyBrush, _, _ = procCreateSolidBrush.Call(colorKey)
	boardBrush, _, _ = procCreateSolidBrush.Call(colBoard)
	fontHUD = mkFontEx(-18, 500, "Microsoft YaHei UI", 1, clearTypeQual)
	fontTitle = mkFontEx(-20, 600, "Microsoft YaHei UI", 1, clearTypeQual)
	fontBody = mkFontEx(-16, 400, "Microsoft YaHei UI", 1, clearTypeQual)
	fontBig = mkFontEx(-40, 700, "Microsoft YaHei UI", 1, clearTypeQual)
	fontHuge = mkFontEx(-52, 700, "Microsoft YaHei UI", 1, clearTypeQual)
	fontSmall = mkFontEx(-13, 400, "Microsoft YaHei UI", 1, clearTypeQual)

	if !registerClass("DeskHUD", syscall.NewCallback(hudProc), syscall.Handle(keyBrush)) ||
		!registerClass("DeskHUDDash", syscall.NewCallback(dashProc), syscall.Handle(boardBrush)) ||
		!registerClass("DeskHUDPanel", syscall.NewCallback(deskProc), syscall.Handle(keyBrush)) {
		alert(appTitle, "窗口类注册失败")
		return
	}

	x, y := defaultHUDPos()
	hwnd, _, err := procCreateWindowExW.Call(wsExLayered|wsExTopmost|wsExNoActivate|wsExToolwindow|wsExTransparent, uintptr(unsafe.Pointer(utf16Ptr("DeskHUD"))), uintptr(unsafe.Pointer(utf16Ptr("DeskHUD"))),
		wsPopup, uintptr(x), uintptr(y), uintptr(hudW), uintptr(hudH), 0, 0, hInst, 0)
	if hwnd == 0 {
		alert(appTitle, "悬浮字创建失败: "+err.Error())
		return
	}
	hwndHUD = hwnd
	applyHUDAlpha()
	procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(hudW), uintptr(hudH), swpNoActivate)
	if cfg.ShowBar {
		procShowWindow.Call(hwnd, swShowNA)
		paintHUD(hwnd)
	}
	createDeskWindow()
	addTray(hwnd)
	openDash()
	if hwndDash != 0 {
		applyWindowIcon(hwndDash)
	}
	if hasCookie() {
		snapMu.Lock()
		snap.Status = "正在读取用量…"
		snapMu.Unlock()
		refreshCursorAsync()
	} else {
		snapMu.Lock()
		snap.Status = "等待粘贴 Cookie"
		snapMu.Unlock()
	}

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		n := 0
		for range t.C {
			n++
			procPostMessageW.Call(hwndHUD, wmAppRefresh, 0, 0)
			if n%180 == 0 && hasCookie() {
				refreshCursorAsync()
			}
		}
	}()
	go startHeartbeat()

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 { break }
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		if pendingMove {
			pendingMove = false
			if dragHWND == hwndHUD && hwndHUD != 0 {
				procSetWindowPos.Call(hwndHUD, hwndTopmost, uintptr(moveX), uintptr(moveY), 0, 0, swpNoSize|swpNoActivate)
				applyHUDAlpha()
			} else if hwndDesk != 0 {
				x, y := moveX, moveY
				if parent, _, _ := procGetParent.Call(hwndDesk); parent != 0 {
					pt := point{x, y}
					procScreenToClient.Call(parent, uintptr(unsafe.Pointer(&pt)))
					x, y = pt.x, pt.y
				}
				procSetWindowPos.Call(hwndDesk, 1, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoActivate|0x0004)
			}
		}
		if pendingUninstall { pendingUninstall = false; launchUninstall() }
		if pendingNoUpdate {
			updateMu.Lock()
			pendingNoUpdate = false
			updateMu.Unlock()
			alertInfo(appTitle, "已是最新版本  v"+appVersion)
		}
		if pendingUpdate { handleUpdatePrompt() }
		if pendingOpen { pendingOpen = false; openDash() }
		if pendingTray { pendingTray = false; showTrayMenu() }
		if pendingSlide {
			pendingSlide = false
			if slideID != 0 {
				applyClick(slideID)
			}
			if pendingSlideEnd {
				pendingSlideEnd = false
				slideID = 0
			}
		}
		if pendingClick {
			pendingClick = false
			applyClick(hitTest(clickX, clickY))
		}
		if pendingColor { pendingColor = false; pickColor() }
		if pendingCookie { pendingCookie = false; applyClipboardCookie() }
		if pendingHUD {
			pendingHUD = false
			refreshStats()
			if cfg.ShowBar {
				applyHUDAlpha()
				paintHUD(hwndHUD)
			}
			if cfg.ShowDesk {
				pendingDesk = true
			}
		}
		if pendingDesk && hwndDesk != 0 {
			pendingDesk = false
			applyDeskAlpha()
			paintDesk(hwndDesk)
		}
		if pendingDash && hwndDash != 0 {
			pendingDash = false
			paintDash(hwndDash)
		}
	}
	sendHeartbeatOnce("offline", 2*time.Second)
	for _, f := range []uintptr{fontHUD, fontTitle, fontBody, fontBig, fontHuge, fontSmall, keyBrush, boardBrush} {
		if f != 0 { procDeleteObject.Call(f) }
	}
}
