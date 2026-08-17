package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const (
	panelW, panelH int32 = 368, 340
	colHudCard           = 0x00261812
	colHudEdge           = 0x00443024
	icoPlan              = 1
	icoCPU               = 2
	icoMem               = 3
	icoCursor            = 4
	icoPrem              = 5
)

func measureStr(hdc uintptr, s string, fnt uintptr) int32 {
	if fnt != 0 {
		procSelectObject.Call(hdc, fnt)
	}
	u, _ := syscall.UTF16FromString(s)
	var sz struct{ cx, cy int32 }
	procGetTextExtentPoint32W.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&sz)))
	return sz.cx
}

var lastHUDKey string
var lastHUDAlpha byte = 0
var lastDeskAlpha byte = 0

func applyHUDAlpha() {
	if hwndHUD == 0 || !cfg.ShowBar {
		return
	}
	v := clampAlphaVal(cfg.BarAlpha)
	a := v * 255 / 100
	if a < 30*255/100 {
		a = 30 * 255 / 100
	}
	if a > 255 {
		a = 255
	}
	b := byte(a)
	if b == lastHUDAlpha {
		return
	}
	lastHUDAlpha = b
	procSetLayeredWindowAttr.Call(hwndHUD, colorKey, uintptr(b), lwaColorKey|lwaAlpha)
}

func ensureLayered(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	if style&wsExLayered == 0 {
		procSetWindowLongPtrW.Call(hwnd, gwlExStyle, style|wsExLayered)
	}
}

func applyDeskAlpha() {
	if hwndDesk == 0 || !cfg.ShowDesk {
		return
	}
	ensureLayered(hwndDesk)
	v := clampAlphaVal(cfg.DeskAlpha)
	b := byte(v * 255 / 100)
	if b == lastDeskAlpha {
		return
	}
	lastDeskAlpha = b
	procSetLayeredWindowAttr.Call(hwndDesk, colorKey, uintptr(b), lwaColorKey|lwaAlpha)
}


func hudFrameKey() string {
	s := currentSnap()
	return fmt.Sprintf("%d|%d|%.1f|%.1f|%t|%s|%d|%d|%t|%t|%t|%d",
		cpuPercent, memLoad, usedGB, totalGB, s.HasData, s.Plan, s.RemainingPct, s.OtherPct,
		cfg.ShowCPU, cfg.ShowMem, cfg.ShowCursor, cfg.Color)
}

func paintHUD(hwnd uintptr) {
	if hwnd == 0 || !cfg.ShowBar {
		return
	}
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return
	}
	paintHUDTo(hwnd, hdc, false)
	procReleaseDC.Call(hwnd, hdc)
}

func paintHUDTo(hwnd, hdc uintptr, force bool) {
	if hwnd == 0 || hdc == 0 || !cfg.ShowBar {
		return
	}
	key := hudFrameKey()
	if !force && key == lastHUDKey {
		return
	}
	s := currentSnap()
	fg := safeColor()
	type cell struct {
		lab, val string
		col      uint32
		ico      int
	}
	var cells []cell
	if s.HasData && s.Plan != "" {
		cells = append(cells, cell{"会员版本", s.Plan, fg, icoPlan})
	} else {
		cells = append(cells, cell{"会员版本", "未连接", colMute, icoPlan})
	}
	if cfg.ShowCPU {
		cells = append(cells, cell{"CPU", fmt.Sprintf("%d%%", cpuPercent), fg, icoCPU})
	}
	if cfg.ShowMem {
		cells = append(cells, cell{"内存", fmt.Sprintf("%d%%  %.1f/%.1f", memLoad, usedGB, totalGB), fg, icoMem})
	}
	if cfg.ShowCursor {
		cv, pv := "—", "—"
		if s.HasData {
			cv = fmt.Sprintf("%d%%", s.RemainingPct)
			if s.OtherPct > 0 {
				n := 100 - s.OtherPct
				if n < 0 {
					n = 0
				}
				pv = fmt.Sprintf("%d%%", n)
			}
		}
		cells = append(cells, cell{"Cursor 模型", cv, fg, icoCursor})
		cells = append(cells, cell{"高级模型", pv, fg, icoPrem})
	}
	gap, h := int32(8), int32(34)
	widths := make([]int32, len(cells))
	total := int32(8)
	for i, c := range cells {
		w := measureStr(hdc, c.lab, fontSmall) + measureStr(hdc, c.val, fontHUD) + 50
		if w < 110 {
			w = 110
		}
		widths[i] = w
		total += w + gap
	}
	needW, needH := total, h+8
	var rc rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	if needW != rc.right-rc.left || needH != rc.bottom-rc.top {
		var wr rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
		cx, _, _ := procGetSystemMetrics.Call(smCxScreen)
		x := wr.left
		if x+needW > int32(cx)-8 {
			x = int32(cx) - needW - 8
		}
		if x < 8 {
			x = 8
		}
		y := wr.top
		if y < 8 {
			y = 8
		}
		procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(needW), uintptr(needH), swpNoActivate)
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	}
	cw, ch := rc.right-rc.left, rc.bottom-rc.top
	if cw < 1 || ch < 1 {
		return
	}
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
	procFillRect.Call(mem, uintptr(unsafe.Pointer(&rc)), keyBrush)
	x := int32(4)
	for i, c := range cells {
		w := widths[i]
		fillRound(mem, x, 4, w, h, 8, colHudCard)
		strokeRound(mem, x, 4, w, h, 8, colHudEdge)
		drawEmbedIcon(mem, x+8, 13, 16, c.ico)
		lw := measureStr(mem, c.lab, fontSmall)
		drawStr(mem, x+28, 4, lw+4, h, c.lab, colMute, fontSmall, dtLeft|dtVCenter)
		drawStr(mem, x+32+lw, 4, w-lw-40, h, c.val, c.col, fontHUD, dtLeft|dtVCenter)
		x += w + gap
	}
	const srcCopy = 0x00CC0020
	procBitBlt.Call(hdc, 0, 0, uintptr(cw), uintptr(ch), mem, 0, 0, srcCopy)
	procSelectObject.Call(mem, old)
	procDeleteObject.Call(bmp)
	procDeleteDC.Call(mem)
	lastHUDKey = key
}

func mixBGR(a, b uint32, t float64) uint32 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab := float64(a&0xFF), float64((a>>8)&0xFF), float64((a>>16)&0xFF)
	br, bg, bb := float64(b&0xFF), float64((b>>8)&0xFF), float64((b>>16)&0xFF)
	u := 1 - t
	return uint32(ar*u+br*t+0.5) | uint32(ag*u+bg*t+0.5)<<8 | uint32(ab*u+bb*t+0.5)<<16
}

func coverEdge(dist, aa float64) float64 {
	x := dist/aa + 0.5
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	return x * x * (3 - 2*x)
}

func drawRing(hdc uintptr, cx, cy, r, thick int32, pct int, fill, track uint32) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	pad := int32(4)
	side := int(r*2 + pad*2)
	if side < 8 {
		return
	}
	type bmih struct {
		size          uint32
		width, height int32
		planes, bits  uint16
		compression   uint32
		imgSize       uint32
		xPpm, yPpm    int32
		used, imp     uint32
	}
	var bits unsafe.Pointer
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdc)
	if hdcMem == 0 {
		return
	}
	hdr := bmih{size: 40, width: int32(side), height: -int32(side), planes: 1, bits: 32}
	hbm, _, _ := procCreateDIBSection.Call(hdcMem, uintptr(unsafe.Pointer(&hdr)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		procDeleteDC.Call(hdcMem)
		return
	}
	old, _, _ := procSelectObject.Call(hdcMem, hbm)
	pix := unsafe.Slice((*uint32)(bits), side*side)
	ocx := float64(r + pad)
	ocy := float64(r + pad)
	outer := float64(r) - 0.35
	inner := float64(r-thick) + 0.35
	if inner < 1 {
		inner = 1
	}
	aa := 1.35
	target := float64(pct) / 100 * 2 * math.Pi
	twoPi := 2 * math.Pi
	for y := 0; y < side; y++ {
		fy := float64(y) + 0.5
		for x := 0; x < side; x++ {
			fx := float64(x) + 0.5
			dx, dy := fx-ocx, fy-ocy
			d := math.Hypot(dx, dy)
			ring := coverEdge(outer-d, aa) * (1 - coverEdge(inner-d, aa))
			if ring <= 0.004 {
				pix[y*side+x] = 0
				continue
			}
			fillT := 0.0
			if pct >= 100 {
				fillT = 1
			} else if pct > 0 {
				ang := math.Atan2(dx, -dy)
				if ang < 0 {
					ang += twoPi
				}
				var sd float64
				if ang <= target {
					sd = math.Min(ang, target-ang) * d
				} else {
					sd = -math.Min(ang-target, twoPi-ang) * d
				}
				fillT = coverEdge(sd, aa)
			}
			col := mixBGR(track, fill, fillT)
			rf := float64(col & 0xFF)
			gf := float64((col >> 8) & 0xFF)
			bf := float64((col >> 16) & 0xFF)
			a := uint32(ring*255 + 0.5)
			pix[y*side+x] = uint32(bf*ring+0.5) | uint32(gf*ring+0.5)<<8 | uint32(rf*ring+0.5)<<16 | a<<24
		}
	}
	const acSrcOver, acSrcAlpha = 0, 1
	blend := uint32(acSrcOver) | uint32(255)<<16 | uint32(acSrcAlpha)<<24
	procAlphaBlend.Call(hdc, uintptr(cx-r-pad), uintptr(cy-r-pad), uintptr(side), uintptr(side), hdcMem, 0, 0, uintptr(side), uintptr(side), uintptr(blend))
	procSelectObject.Call(hdcMem, old)
	procDeleteObject.Call(hbm)
	procDeleteDC.Call(hdcMem)
}

func paintDesk(hwnd uintptr) {
	if hwnd == 0 || !cfg.ShowDesk {
		return
	}
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return
	}
	var rc rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), keyBrush)
	fillRound(hdc, 2, 2, rc.right-4, rc.bottom-4, 16, colCard)
	strokeRound(hdc, 2, 2, rc.right-4, rc.bottom-4, 16, colCyan)
	s := currentSnap()
	fg := safeColor()
	drawStr(hdc, 14, 10, 180, 16, "XT-CURSOR  v"+appVersion, colCyan, fontSmall, dtLeft)
	drawStr(hdc, 220, 10, 130, 16, time.Now().Format("15:04:05"), colMute, fontSmall, dtRight|dtVCenter)

	rp, prem := 0, 0
	if s.HasData {
		rp = s.RemainingPct
		if s.OtherPct > 0 {
			prem = 100 - s.OtherPct
			if prem < 0 {
				prem = 0
			}
		}
	}
	drawRing(hdc, 92, 88, 44, 8, rp, colCyan, colTrack)
	txt := "—"
	if s.HasData {
		txt = fmt.Sprintf("%d%%", rp)
	}
	drawStr(hdc, 56, 74, 72, 28, txt, fg, fontTitle, dtCenter|dtVCenter)
	drawStr(hdc, 40, 140, 104, 16, "Cursor 模型", colMute, fontSmall, dtCenter)

	drawRing(hdc, 276, 88, 44, 8, prem, colLilac, colTrack)
	pt := "—"
	if s.HasData && s.OtherPct > 0 {
		pt = fmt.Sprintf("%d%%", prem)
	}
	drawStr(hdc, 240, 74, 72, 28, pt, fg, fontTitle, dtCenter|dtVCenter)
	drawStr(hdc, 224, 140, 104, 16, "高级模型", colMute, fontSmall, dtCenter)

	drawStr(hdc, 16, 168, 50, 16, "CPU", colMute, fontSmall, dtLeft)
	drawStr(hdc, 280, 168, 72, 16, fmt.Sprintf("%d%%", cpuPercent), fg, fontSmall, dtRight)
	bar(hdc, 16, 186, 336, 8, cpuPercent, colOrange)

	drawStr(hdc, 16, 204, 50, 16, "内存", colMute, fontSmall, dtLeft)
	drawStr(hdc, 200, 204, 152, 16, fmt.Sprintf("%.1f / %.1f GB", usedGB, totalGB), fg, fontSmall, dtRight)
	bar(hdc, 16, 222, 336, 8, memLoad, colGreen)

	drawStr(hdc, 16, 244, 80, 16, "Grok", colMute, fontSmall, dtLeft)
	if s.HasData && isUltraPlan(s.Plan) {
		drawStr(hdc, 280, 244, 72, 16, fmt.Sprintf("%d%%", s.GrokPct), fg, fontSmall, dtRight)
		bar(hdc, 16, 262, 336, 8, s.GrokPct, colOrange)
		drawStr(hdc, 16, 276, 336, 16, s.GrokReset, colMute, fontSmall, dtLeft)
	} else {
		drawStr(hdc, 16, 262, 336, 20, "升级 Ultra 后可查看 Grok 用量", colMute, fontSmall, dtLeft)
	}

	plan := "未连接"
	if s.HasData && s.Plan != "" {
		plan = s.Plan
	}
	drawEmbedIcon(hdc, 16, 308, 16, icoPlan)
	drawStr(hdc, 36, 306, 316, 20, "会员版本 "+plan+"  ·  拖动可移动", colMute, fontSmall, dtLeft|dtVCenter)
	procReleaseDC.Call(hwnd, hdc)
}

func loadDeskPos() (int32, int32, bool) {
	b, err := os.ReadFile(filepath.Join(appDir(), "pos_desk.json"))
	if err != nil {
		return 0, 0, false
	}
	var p posFile
	if json.Unmarshal(b, &p) != nil {
		return 0, 0, false
	}
	return p.X, p.Y, true
}

func saveDeskPos(hwnd uintptr) {
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	_ = os.MkdirAll(appDir(), 0755)
	b, _ := json.Marshal(posFile{X: r.left, Y: r.top})
	_ = os.WriteFile(filepath.Join(appDir(), "pos_desk.json"), b, 0644)
}

func defaultDeskPos() (int32, int32) {
	cx, _, _ := procGetSystemMetrics.Call(smCxScreen)
	cy, _, _ := procGetSystemMetrics.Call(smCyScreen)
	if x, y, ok := loadDeskPos(); ok && x >= 0 && y >= 0 && x < int32(cx)-40 && y < int32(cy)-40 {
		return x, y
	}
	x := int32(cx) - panelW - 36
	if x < 16 {
		x = 16
	}
	return x, 72
}

func applyOverlayVis() {
	if hwndHUD != 0 {
		if cfg.ShowBar {
			procShowWindow.Call(hwndHUD, swShowNA)
			pendingHUD = true
		} else {
			procShowWindow.Call(hwndHUD, swHide)
		}
	}
	if hwndDesk != 0 {
		if cfg.ShowDesk {
			procShowWindow.Call(hwndDesk, swShowNA)
			pendingDesk = true
		} else {
			procShowWindow.Call(hwndDesk, swHide)
		}
	}
}

func findIconLayer() uintptr {
	progman, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(utf16Ptr("Progman"))), 0)
	if progman != 0 {
		v, _, _ := procFindWindowExW.Call(progman, 0, uintptr(unsafe.Pointer(utf16Ptr("SHELLDLL_DefView"))), 0)
		if v != 0 {
			return v
		}
	}
	foundDefView = 0
	procEnumWindows.Call(enumDefViewCb, 0)
	return foundDefView
}

func attachToDesktop(hwnd uintptr) {
	v := findIconLayer()
	if v == 0 {
		return
	}
	var wr rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	procSetParent.Call(hwnd, v)
	pt := point{wr.left, wr.top}
	procScreenToClient.Call(v, uintptr(unsafe.Pointer(&pt)))
	if pt.x < 8 {
		pt.x = 8
	}
	if pt.y < 8 {
		pt.y = 8
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(pt.x), uintptr(pt.y), uintptr(panelW), uintptr(panelH), swpNoActivate)
}

func createDeskWindow() {
	x, y := defaultDeskPos()
	hwnd, _, _ := procCreateWindowExW.Call(wsExLayered|wsExNoActivate|wsExToolwindow, uintptr(unsafe.Pointer(utf16Ptr("DeskHUDPanel"))), uintptr(unsafe.Pointer(utf16Ptr(appTitle))),
		wsPopup|wsVisible, uintptr(x), uintptr(y), uintptr(panelW), uintptr(panelH), 0, 0, hInst, 0)
	if hwnd == 0 {
		return
	}
	hwndDesk = hwnd
	attachToDesktop(hwnd)
	ensureLayered(hwnd)
	lastDeskAlpha = 0
	applyDeskAlpha()
	if cfg.ShowDesk {
		procShowWindow.Call(hwnd, swShowNA)
		paintDesk(hwnd)
		pendingDesk = true
	} else {
		procShowWindow.Call(hwnd, swHide)
	}
}

func deskProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmPaint:
		var ps paintStruct
		procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		pendingDesk = true
		return 0
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
			saveDeskPos(hwnd)
		}
		return 0
	case wmDestroy:
		saveDeskPos(hwnd)
		hwndDesk = 0
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}
