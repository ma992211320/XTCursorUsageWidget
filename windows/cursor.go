package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ModelRow struct {
	Name   string
	Tokens float64
	Amount float64
	Group  string
	In, Out, CacheR, CacheW float64
	EstUSD float64
	HasPrice bool
}

type tokParts struct {
	in, out, cacheR, cacheW float64
}

func (p tokParts) total() float64 { return p.in + p.out + p.cacheR + p.cacheW }

type UsageSnap struct {
	HasData      bool
	Plan         string
	RemainingPct int
	UsedPct      int
	CursorPct    int
	OtherPct     int
	LimitUSD     float64
	UsedUSD      float64
	OverUSD      float64
	CursorUSD    float64
	OtherUSD     float64
	HasCursorUSD bool
	HasOtherUSD  bool
	TotalTokens  float64
	InputTokens  float64
	OutputTokens float64
	TodayIn, TodayOut, TodayCacheR, TodayCacheW float64
	HasToday     bool
	DisplayMsg   string
	CycleEnd     string
	GrokPct      int
	GrokReset    string
	Status       string
	Models       []ModelRow
}

var (
	snap   UsageSnap
	snapMu sync.Mutex
	httpC  = &http.Client{Timeout: 15 * time.Second}
)

func sessionPath() string { return filepath.Join(appDir(), "session") }

func normalizeCookie(raw string) string {
	text := strings.TrimSpace(raw)
	low := strings.ToLower(text)
	if i := strings.Index(low, "workoscursorsessiontoken="); i >= 0 {
		text = text[i+len("WorkosCursorSessionToken="):]
		if j := strings.IndexAny(text, ";\r\n"); j >= 0 {
			text = text[:j]
		}
	}
	text = strings.TrimSpace(text)
	return strings.ReplaceAll(text, "%3A%3A", "::")
}

func saveCookie(raw string) error {
	v := normalizeCookie(raw)
	if v == "" {
		return errors.New("请粘贴 Cookie")
	}
	_ = os.MkdirAll(appDir(), 0700)
	return os.WriteFile(sessionPath(), []byte(v), 0600)
}

func readCookie() string {
	b, err := os.ReadFile(sessionPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func deleteCookie() { _ = os.Remove(sessionPath()) }

func hasCookie() bool { return readCookie() != "" }

func cursorJSON(method, path string, body any) (map[string]any, int, error) {
	token := readCookie()
	if token == "" {
		return nil, 0, errors.New("还没有登录态。请先粘贴 Cookie。")
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "https://cursor.com"+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://cursor.com")
	req.Header.Set("Referer", "https://cursor.com/dashboard/usage")
	req.Header.Set("Cookie", "WorkosCursorSessionToken="+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpC.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 204 {
		return nil, resp.StatusCode, errors.New("Cookie 已失效，请重新粘贴")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, resp.StatusCode, fmt.Errorf("读取 %s 失败 (%d)", path, resp.StatusCode)
	}
	if len(data) == 0 {
		return nil, resp.StatusCode, fmt.Errorf("读取 %s 失败", path)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, resp.StatusCode, err
	}
	return obj, resp.StatusCode, nil
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}
func asArr(v any) []any {
	a, _ := v.([]any)
	return a
}
func asNum(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		re := regexp.MustCompile(`[^0-9.-]`)
		var f float64
		fmt.Sscanf(re.ReplaceAllString(t, ""), "%f", &f)
		return f
	}
	return 0
}
func asStr(v any) string {
	s, _ := v.(string)
	return s
}
func asInt(v any) int { return int(asNum(v) + 0.5) }

func firstNum(m map[string]any, keys ...string) float64 {
	if m == nil {
		return 0
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if n := asNum(v); n != 0 {
				return n
			}
		}
	}
	return 0
}

func sumTokenFields(m map[string]any) float64 {
	n := firstNum(m, "inputTokens", "totalInputTokens") +
		firstNum(m, "outputTokens", "totalOutputTokens") +
		firstNum(m, "cacheReadTokens", "totalCacheReadTokens") +
		firstNum(m, "cacheWriteTokens", "totalCacheWriteTokens")
	if n <= 0 {
		n = firstNum(m, "totalTokens", "tokens")
	}
	return n
}

func readTok(m map[string]any) tokParts {
	if m == nil {
		return tokParts{}
	}
	return tokParts{
		in:     firstNum(m, "inputTokens", "totalInputTokens"),
		out:    firstNum(m, "outputTokens", "totalOutputTokens"),
		cacheR: firstNum(m, "cacheReadTokens", "totalCacheReadTokens"),
		cacheW: firstNum(m, "cacheWriteTokens", "totalCacheWriteTokens"),
	}
}

func tokenPartsOf(o map[string]any) tokParts {
	if o == nil {
		return tokParts{}
	}
	p := readTok(asMap(o["tokenUsage"]))
	if p.total() > 0 {
		return p
	}
	return readTok(o)
}

func tokensFrom(o map[string]any) float64 {
	p := tokenPartsOf(o)
	if n := p.total(); n > 0 {
		return n
	}
	return sumTokenFields(o)
}

func formatMoney(n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("$%.0f", n)
	}
	return fmt.Sprintf("$%.2f", n)
}

func formatMoneyPair(used, limit float64) string {
	return formatMoney(used) + " / " + formatMoney(limit)
}

func percentFromText(s string) int {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	var f float64
	fmt.Sscanf(m[1], "%f", &f)
	return int(f + 0.5)
}

func modelGroup(name string) string {
	k := strings.ToLower(name)
	if strings.HasPrefix(k, "cursor-") || strings.HasPrefix(k, "composer-") || k == "auto" {
		return "cursor"
	}
	return "other"
}

func walkJSON(v any, fn func(string, any)) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			fn(k, child)
			walkJSON(child, fn)
		}
	case []any:
		for _, child := range t {
			walkJSON(child, fn)
		}
	}
}

func grokFromMaps(roots ...map[string]any) (pct int, reset string) {
	for _, root := range roots {
		if root == nil {
			continue
		}
		walkJSON(root, func(key string, value any) {
			name := strings.ToLower(key)
			grokish := strings.Contains(name, "grok")
			weekly := strings.Contains(name, "weekly") || strings.Contains(name, "week")
			botish := strings.Contains(name, "bot") && (grokish || weekly)
			if pct == 0 && (grokish || botish) {
				if n := asInt(value); n > 0 {
					pct = n
				} else if n := percentFromText(asStr(value)); n > 0 {
					pct = n
				}
			}
			if o := asMap(value); o != nil && (grokish || botish || weekly) {
				if pct == 0 {
					pct = asInt(o["percent"])
					if pct == 0 {
						pct = asInt(o["usedPercent"])
					}
					if pct == 0 {
						pct = percentFromText(asStr(o["displayMessage"]))
					}
				}
				if reset == "" {
					reset = asStr(o["resetAt"])
					if reset == "" {
						reset = asStr(o["resetsAt"])
					}
					if reset == "" {
						reset = asStr(o["resetDate"])
					}
				}
			}
			if reset == "" && strings.Contains(name, "grok") && strings.Contains(name, "reset") {
				reset = asStr(value)
			}
		})
	}
	return
}

func formatGrokReset(reset string) string {
	if reset == "" {
		return "刷新时间未返回"
	}
	var tm time.Time
	if n := asNum(reset); n > 1e12 {
		tm = time.UnixMilli(int64(n))
	} else if n > 1e9 {
		tm = time.Unix(int64(n), 0)
	} else {
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"} {
			if p, err := time.Parse(layout, reset); err == nil {
				tm = p
				break
			}
		}
	}
	if tm.IsZero() {
		return reset
	}
	label := tm.In(time.Local).Format("1月2日")
	days := int(tm.In(time.Local).Sub(time.Now()).Hours()/24 + 0.5)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("刷新 %s（还剩 %d 天）", label, days)
}

func weeklyResetFrom(startMs float64) string {
	if startMs <= 0 {
		return ""
	}
	reset := time.UnixMilli(int64(startMs)).Add(7 * 24 * time.Hour)
	now := time.Now()
	for !reset.After(now) {
		reset = reset.Add(7 * 24 * time.Hour)
	}
	return reset.Format(time.RFC3339)
}

func formatWan(n float64) string {
	if n >= 10000 {
		return fmt.Sprintf("%.1f万", n/10000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", n/1000)
	}
	if n <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f", n)
}

func fetchEventPage(userID, start, end float64, page int) map[string]any {
	bodies := []map[string]any{
		{"teamId": 0, "userId": userID, "startDate": fmt.Sprintf("%.0f", start), "endDate": fmt.Sprintf("%.0f", end), "page": page, "pageSize": 200},
		{"userId": userID, "startDate": fmt.Sprintf("%.0f", start), "endDate": fmt.Sprintf("%.0f", end), "page": page, "pageSize": 200},
	}
	for _, body := range bodies {
		data, _, e := cursorJSON("POST", "/api/dashboard/get-filtered-usage-events", body)
		if e == nil && data != nil {
			return data
		}
	}
	return nil
}

func fetchTodayTokens(userID float64) tokParts {
	now := time.Now()
	start := float64(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UnixMilli())
	end := float64(now.UnixMilli())
	var sum tokParts
	for page := 1; page <= 2; page++ {
		data := fetchEventPage(userID, start, end, page)
		if data == nil {
			break
		}
		batch := asArr(data["usageEventsDisplay"])
		if len(batch) == 0 {
			batch = asArr(data["usageEvents"])
		}
		if len(batch) == 0 {
			break
		}
		for _, row := range batch {
			p := tokenPartsOf(asMap(row))
			sum.in += p.in
			sum.out += p.out
			sum.cacheR += p.cacheR
			sum.cacheW += p.cacheW
		}
		total := asInt(data["totalUsageEventsCount"])
		if total > 0 && page*200 >= total {
			break
		}
	}
	return sum
}

func sumGroupUSD(models []ModelRow, group string) (float64, bool) {
	var n float64
	ok := false
	for _, m := range models {
		if m.Group != group {
			continue
		}
		if m.HasPrice {
			n += m.EstUSD
			ok = true
		} else if m.Amount > 0 {
			n += m.Amount
			ok = true
		}
	}
	return n, ok
}

func fetchUsage() (UsageSnap, error) {
	me, _, err := cursorJSON("GET", "/api/auth/me", nil)
	if err != nil {
		return UsageSnap{}, err
	}
	userID := asNum(me["id"])
	if userID == 0 {
		return UsageSnap{}, errors.New("Cookie 已失效，请重新粘贴")
	}

	period, _, _ := cursorJSON("POST", "/api/dashboard/get-current-period-usage", map[string]any{})
	plan, _, _ := cursorJSON("POST", "/api/dashboard/get-plan-info", map[string]any{})
	nowMs := float64(time.Now().UnixMilli())
	t := time.Now()
	monthStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).UnixMilli()

	var aggregated map[string]any
	for _, body := range []map[string]any{
		{"teamId": 0, "userId": userID, "startDate": fmt.Sprintf("%.0f", float64(monthStart)), "endDate": fmt.Sprintf("%.0f", nowMs)},
		{"teamId": -1, "userId": userID, "startDate": fmt.Sprintf("%.0f", float64(monthStart)), "endDate": fmt.Sprintf("%.0f", nowMs)},
		{"startDate": fmt.Sprintf("%.0f", float64(monthStart)), "endDate": fmt.Sprintf("%.0f", nowMs)},
	} {
		data, _, e := cursorJSON("POST", "/api/dashboard/get-aggregated-usage-events", body)
		if e == nil && data != nil && (data["aggregations"] != nil || data["totalInputTokens"] != nil) {
			aggregated = data
			break
		}
	}

	if period == nil && aggregated == nil {
		return UsageSnap{}, errors.New("接口没有返回用量字段")
	}

	planUsage := asMap(nil)
	if period != nil {
		planUsage = asMap(period["planUsage"])
	}
	planInfo := asMap(nil)
	if plan != nil {
		planInfo = asMap(plan["planInfo"])
	}
	usedCents := asNum(planUsage["includedSpend"])
	if usedCents == 0 {
		usedCents = asNum(planUsage["used"])
	}
	if usedCents == 0 {
		usedCents = asNum(planUsage["totalSpend"])
	}
	limitCents := asNum(planUsage["limit"])
	if limitCents == 0 {
		limitCents = asNum(planInfo["includedAmountCents"])
	}
	msg := ""
	if period != nil {
		msg = asStr(period["displayMessage"])
	}
	usedPct := 0
	if limitCents > 0 {
		usedPct = int((usedCents/limitCents)*100 + 0.5)
	} else {
		usedPct = percentFromText(msg)
	}
	cursorPct := asInt(planUsage["autoPercentUsed"])
	if cursorPct == 0 && period != nil {
		cursorPct = percentFromText(asStr(period["autoModelSelectedDisplayMessage"]))
	}
	otherPct := asInt(planUsage["apiPercentUsed"])
	if otherPct == 0 && period != nil {
		otherPct = percentFromText(asStr(period["namedModelSelectedDisplayMessage"]))
	}
	if otherPct == 0 && usedPct > 0 {
		otherPct = usedPct
	}
	remain := 100 - usedPct
	if cursorPct > 0 {
		remain = 100 - cursorPct
		usedPct = cursorPct
	}
	if remain < 0 {
		remain = 0
	}
	if remain > 100 {
		remain = 100
	}

	planName := asStr(planInfo["planName"])
	if planName == "" && period != nil {
		planName = asStr(period["membershipType"])
	}
	if planName == "" {
		planName = "个人"
	}

	in, out := 0.0, 0.0
	if aggregated != nil {
		in = firstNum(aggregated, "totalInputTokens", "inputTokens")
		out = firstNum(aggregated, "totalOutputTokens", "outputTokens")
		if in+out == 0 {
			if tu := asMap(aggregated["tokenUsage"]); tu != nil {
				in = firstNum(tu, "inputTokens", "totalInputTokens")
				out = firstNum(tu, "outputTokens", "totalOutputTokens")
			}
		}
	}
	total := in + out
	var models []ModelRow
	if aggregated != nil {
		for _, row := range asArr(aggregated["aggregations"]) {
			o := asMap(row)
			if o == nil {
				continue
			}
			name := asStr(o["modelIntent"])
			if name == "" {
				name = asStr(o["model"])
			}
			if name == "" {
				name = "unknown"
			}
			p := tokenPartsOf(o)
			tok := p.total()
			if tok <= 0 {
				tok = tokensFrom(o)
			}
			cents := firstNum(o, "totalCents", "cents")
			if tok <= 0 && cents <= 0 {
				continue
			}
			est, has := estimateUSD(name, p.in, p.out, p.cacheR, p.cacheW)
			models = append(models, ModelRow{
				Name: name, Tokens: tok, Amount: cents / 100, Group: modelGroup(name),
				In: p.in, Out: p.out, CacheR: p.cacheR, CacheW: p.cacheW,
				EstUSD: est, HasPrice: has,
			})
			if total == 0 {
				total += tok
			}
		}
		for i := 0; i < len(models); i++ {
			for j := i + 1; j < len(models); j++ {
				if models[j].Tokens > models[i].Tokens || (models[j].Tokens == models[i].Tokens && models[j].Amount > models[i].Amount) {
					models[i], models[j] = models[j], models[i]
				}
			}
		}
	}

	cycleEnd := ""
	if period != nil {
		if n := asNum(period["billingCycleEnd"]); n > 1e12 {
			cycleEnd = time.UnixMilli(int64(n)).Format("1月2日")
		} else if s := asStr(period["billingCycleEnd"]); s != "" {
			cycleEnd = s
		}
	}

	today := fetchTodayTokens(userID)
	cursorUSD, hasCursorUSD := sumGroupUSD(models, "cursor")
	otherUSD, hasOtherUSD := sumGroupUSD(models, "other")
	sand, _, _ := cursorJSON("POST", "/api/dashboard/get-sand-usage-status", map[string]any{})
	grokPct, grokReset := grokFromMaps(period, plan, sand)
	if sand != nil {
		if n := asInt(sand["usagePercent"]); n > 0 {
			grokPct = n
		}
		if n := asInt(sand["usage_percent"]); n > 0 {
			grokPct = n
		}
		if r := asStr(sand["nextResetTimestampUtc"]); r != "" {
			grokReset = r
		} else if r := asStr(sand["next_reset_timestamp_utc"]); r != "" {
			grokReset = r
		}
	}
	if grokReset == "" && period != nil {
		startMs := asNum(period["billingCycleStart"])
		grokReset = weeklyResetFrom(startMs)
	}
	usedUSD := usedCents / 100
	limitUSD := limitCents / 100
	over := 0.0
	if limitUSD > 0 && usedUSD > limitUSD {
		over = usedUSD - limitUSD
	}

	return UsageSnap{
		HasData:      true,
		Plan:         planName,
		RemainingPct: remain,
		UsedPct:      usedPct,
		CursorPct:    cursorPct,
		OtherPct:     otherPct,
		LimitUSD:     limitUSD,
		UsedUSD:      usedUSD,
		OverUSD:      over,
		CursorUSD:    cursorUSD,
		OtherUSD:     otherUSD,
		HasCursorUSD: hasCursorUSD,
		HasOtherUSD:  hasOtherUSD,
		TotalTokens:  total,
		InputTokens:  in,
		OutputTokens: out,
		TodayIn:      today.in,
		TodayOut:     today.out,
		TodayCacheR:  today.cacheR,
		TodayCacheW:  today.cacheW,
		HasToday:     today.total() > 0,
		DisplayMsg:   msg,
		CycleEnd:     cycleEnd,
		GrokPct:      grokPct,
		GrokReset:    formatGrokReset(grokReset),
		Status:       "已更新 " + time.Now().Format("15:04"),
		Models:       models,
	}, nil
}

func refreshCursorAsync() {
	go func() {
		s, err := fetchUsage()
		snapMu.Lock()
		if err != nil {
			if strings.Contains(err.Error(), "失效") {
				deleteCookie()
			}
			snap.Status = err.Error()
			if !snap.HasData {
				snap.HasData = false
			}
		} else {
			s.Status = "已更新 " + time.Now().Format("15:04")
			snap = s
		}
		snapMu.Unlock()
		if hwndHUD != 0 {
			procPostMessageW.Call(hwndHUD, wmAppRefresh, 0, 0)
		}
	}()
}

func currentSnap() UsageSnap {
	snapMu.Lock()
	defer snapMu.Unlock()
	return snap
}
