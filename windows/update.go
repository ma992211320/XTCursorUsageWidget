package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

const (
	mbYesNo    = 0x00000004
	mbIconInfo = 0x00000040
	idYes      = 6
	idNo       = 7
)

type updateInfo struct {
	Ok        bool   `json:"ok"`
	Latest    string `json:"latest"`
	Title     string `json:"title"`
	Update    bool   `json:"update"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Changelog string `json:"changelog"`
	Size      int64  `json:"size"`
}

var (
	updateMu       sync.Mutex
	pendingUpdate   bool
	pendingNoUpdate bool
	updateOffer     updateInfo
	promptedLatest  string
)

func considerUpdate(info updateInfo) {
	if !info.Update || info.Latest == "" || info.URL == "" {
		return
	}
	if info.Latest == cfg.SkipVersion {
		return
	}
	updateMu.Lock()
	defer updateMu.Unlock()
	if promptedLatest == info.Latest {
		return
	}
	promptedLatest = info.Latest
	updateOffer = info
	pendingUpdate = true
	if hwndHUD != 0 {
		procPostMessageW.Call(hwndHUD, wmAppRefresh, 0, 0)
	}
}

func handleUpdatePrompt() {
	updateMu.Lock()
	info := updateOffer
	pendingUpdate = false
	updateMu.Unlock()
	if !info.Update || info.Latest == "" {
		return
	}
	body := "发现新版本  v" + info.Latest
	if info.Title != "" {
		body += "\n" + info.Title
	}
	if info.Changelog != "" {
		body += "\n\n" + info.Changelog
	}
	body += "\n\n是：立即更新并打开安装程序\n否：忽略本次更新"
	owner := hwndDash
	r, _, _ := procMessageBoxW.Call(owner, uintptr(unsafe.Pointer(utf16Ptr(body))), uintptr(unsafe.Pointer(utf16Ptr(appTitle))), mbYesNo|mbIconInfo)
	if r == idNo {
		cfg.SkipVersion = info.Latest
		saveSettings()
		return
	}
	if r != idYes {
		return
	}
	path, err := downloadSetup(info.URL, info.SHA256, info.Latest)
	if err != nil {
		alert(appTitle, "下载更新失败：\n"+err.Error()+"\n\n也可到网站手动下载。")
		return
	}
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	if err := cmd.Start(); err != nil {
		alert(appTitle, "无法打开安装程序：\n"+err.Error())
		return
	}
	if hwndHUD != 0 {
		procDestroyWindow.Call(hwndHUD)
	}
}

func checkUpdateNow() {
	req, err := http.NewRequest(http.MethodGet, reportBase+"/api/update?version="+appVersion, nil)
	if err != nil {
		alert(appTitle, "检查更新失败")
		return
	}
	req.Header.Set("User-Agent", appTitle+"/"+appVersion)
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		alert(appTitle, "检查更新失败：\n"+err.Error())
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var info updateInfo
	if json.Unmarshal(raw, &info) != nil {
		alert(appTitle, "检查更新失败：回包无效")
		return
	}
	if !info.Update {
		updateMu.Lock()
		pendingNoUpdate = true
		updateMu.Unlock()
		if hwndHUD != 0 {
			procPostMessageW.Call(hwndHUD, wmAppRefresh, 0, 0)
		}
		return
	}
	updateMu.Lock()
	promptedLatest = ""
	updateMu.Unlock()
	considerUpdate(info)
}

func downloadSetup(url, wantSHA, ver string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", appTitle+"/"+appVersion)
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}
	path := filepath.Join(os.TempDir(), "XT-cursor-setup-"+ver+".exe")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if wantSHA != "" && sum != wantSHA {
		_ = os.Remove(path)
		return "", fmt.Errorf("文件校验失败")
	}
	return path, nil
}

func alertInfo(t, s string) {
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(s))), uintptr(unsafe.Pointer(utf16Ptr(t))), mbOK|mbIconInfo)
}
