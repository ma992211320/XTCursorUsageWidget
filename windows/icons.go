package main

import (
	"bytes"
	"embed"
	"image/png"
	"sync"
	"unsafe"
)

//go:embed icons/*.png
var iconFS embed.FS

type iconBMP struct {
	hdc, bmp uintptr
	w, h     int32
}

var iconOnce sync.Once
var iconSet map[int]*iconBMP

func iconName(kind int) string {
	switch kind {
	case icoPlan:
		return "crown"
	case icoCPU:
		return "cpu"
	case icoMem:
		return "memory-stick"
	case icoCursor:
		return "refresh-cw"
	case icoPrem:
		return "target"
	default:
		return ""
	}
}

func loadIconPNG(name string) *iconBMP {
	b, err := iconFS.ReadFile("icons/" + name + ".png")
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	r := img.Bounds()
	w, h := int32(r.Dx()), int32(r.Dy())
	if w < 1 || h < 1 {
		return nil
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
	hdcMem, _, _ := procCreateCompatibleDC.Call(0)
	if hdcMem == 0 {
		return nil
	}
	hdr := bmih{size: 40, width: w, height: -h, planes: 1, bits: 32}
	hbm, _, _ := procCreateDIBSection.Call(hdcMem, uintptr(unsafe.Pointer(&hdr)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		procDeleteDC.Call(hdcMem)
		return nil
	}
	procSelectObject.Call(hdcMem, hbm)
	pix := unsafe.Slice((*uint32)(bits), int(w*h))
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			rr, gg, bb, aa := img.At(r.Min.X+x, r.Min.Y+y).RGBA()
			a := aa >> 8
			rf := (rr >> 8) * a / 255
			gf := (gg >> 8) * a / 255
			bf := (bb >> 8) * a / 255
			pix[y*int(w)+x] = bf | gf<<8 | rf<<16 | a<<24
		}
	}
	return &iconBMP{hdc: hdcMem, bmp: hbm, w: w, h: h}
}

func ensureIcons() {
	iconOnce.Do(func() {
		iconSet = map[int]*iconBMP{}
		for _, k := range []int{icoPlan, icoCPU, icoMem, icoCursor, icoPrem} {
			if n := iconName(k); n != "" {
				iconSet[k] = loadIconPNG(n)
			}
		}
	})
}

func drawEmbedIcon(hdc uintptr, x, y, s int32, kind int) {
	if s < 12 {
		s = 12
	}
	ensureIcons()
	ic := iconSet[kind]
	if ic == nil || ic.hdc == 0 {
		return
	}
	const acSrcOver, acSrcAlpha = 0, 1
	blend := uint32(acSrcOver) | uint32(255)<<16 | uint32(acSrcAlpha)<<24
	procAlphaBlend.Call(hdc, uintptr(x), uintptr(y), uintptr(s), uintptr(s), ic.hdc, 0, 0, uintptr(ic.w), uintptr(ic.h), uintptr(blend))
}
