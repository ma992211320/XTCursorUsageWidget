package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	appTitle     = "XT-cursor用量小工具"
	appVersion   = "1.2.4"
	appRegKey    = `Software\Microsoft\Windows\CurrentVersion\Uninstall\XTCursorUsage`
	hkeyCU       = 0x80000001
	keyWrite     = 0x20006
	regSZ        = 1
	regDWORD     = 4
	wmSetIcon    = 0x0080
)

var appIcon uintptr

func exePath() string {
	var buf [32768]uint16
	n, _, _ := procGetModuleFileNameW.Call(hInst, uintptr(unsafe.Pointer(&buf[0])), 32768)
	if n == 0 {
		n, _, _ = procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), 32768)
	}
	if n == 0 {
		return ""
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

func setRegSZ(key uintptr, name, val string) {
	u, _ := syscall.UTF16FromString(val)
	procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(utf16Ptr(name))), 0, regSZ, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)*2))
}

func setRegDW(key uintptr, name string, v uint32) {
	procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(utf16Ptr(name))), 0, regDWORD, uintptr(unsafe.Pointer(&v)), 4)
}

func registerApp() {
	path := exePath()
	if path == "" {
		return
	}
	var hkey uintptr
	var disp uint32
	r, _, _ := procRegCreateKeyExW.Call(hkeyCU, uintptr(unsafe.Pointer(utf16Ptr(appRegKey))), 0, 0, 0, keyWrite, 0, uintptr(unsafe.Pointer(&hkey)), uintptr(unsafe.Pointer(&disp)))
	if r != 0 || hkey == 0 {
		return
	}
	defer procRegCloseKey.Call(hkey)
	setRegSZ(hkey, "DisplayName", appTitle)
	setRegSZ(hkey, "DisplayIcon", path+",0")
	setRegSZ(hkey, "Publisher", "XT")
	setRegSZ(hkey, "InstallLocation", filepath.Dir(path))
	setRegSZ(hkey, "UninstallString", `"`+path+`" --unregister`)
	setRegSZ(hkey, "DisplayVersion", "1.0.0")
	setRegDW(hkey, "NoModify", 1)
	setRegDW(hkey, "NoRepair", 1)
	if fi, err := os.Stat(path); err == nil {
		setRegDW(hkey, "EstimatedSize", uint32(fi.Size()/1024))
	}
}

func unregisterApp() {
	procRegDeleteKeyW.Call(hkeyCU, uintptr(unsafe.Pointer(utf16Ptr(appRegKey))))
}


const (
	gwlExStyle      = ^uintptr(19)
	swpFrameChanged = 0x0020
	swpNoZOrder     = 0x0004
	swpNoMove       = 0x0002
)

func setHUDClickThrough(on bool) {
	if hwndHUD == 0 {
		return
	}
	style, _, _ := procGetWindowLongPtrW.Call(hwndHUD, gwlExStyle)
	if on {
		style |= wsExTransparent
	} else {
		style &^= wsExTransparent
	}
	procSetWindowLongPtrW.Call(hwndHUD, gwlExStyle, style)
	procSetWindowPos.Call(hwndHUD, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

func uninstallerPath() string {
	dir := filepath.Dir(exePath())
	p := filepath.Join(dir, "卸载.exe")
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		return p
	}
	return ""
}

func launchUninstall() {
	p := uninstallerPath()
	if p == "" {
		alert(appTitle, "当前是绿色版，系统设置里没有卸载项。请使用「安装版」安装后，再在本页或 设置 → 应用 里卸载。")
		return
	}
	sendUninstallHeartbeat()
	cmd := exec.Command(p, "--uninstall")
	cmd.Dir = filepath.Dir(p)
	if err := cmd.Start(); err != nil {
		alert(appTitle, "无法打开卸载程序：\n"+err.Error())
		return
	}
	if hwndHUD != 0 {
		procDestroyWindow.Call(hwndHUD)
	}
}
