import AppKit
import SwiftUI

private let boardBackground = Color(red: 0.02, green: 0.12, blue: 0.22)
private let cardBackground = Color(red: 0.02, green: 0.18, blue: 0.31)
private let cursorTint = Color.cyan
private let otherTint = Color(red: 0.73, green: 0.58, blue: 1.0)
private let pagePadding: CGFloat = 22
private let wideBreakpoint: CGFloat = 660

private enum DashboardPage: String, CaseIterable, Identifiable {
    case overview = "概览"
    case models = "模型"
    case settings = "设置"

    var id: String { rawValue }
}

@main
struct CursorUsageWidgetApp: App {
    @StateObject private var store: UsageStore

    init() {
        BackgroundRefresh.runAndExitIfNeeded()
        _store = StateObject(wrappedValue: UsageStore())
    }

    var body: some Scene {
        WindowGroup("Cursor用量") {
            DashboardView(store: store)
                .frame(minWidth: 640, idealWidth: 720, minHeight: 520, idealHeight: 640)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(boardBackground)
                .background(WindowPinProbe(pinned: store.isPinned))
                .onAppear {
                    applyPin(store.isPinned)
                    DispatchQueue.main.async { applyPin(store.isPinned) }
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.4) { applyPin(store.isPinned) }
                }
                .onChange(of: store.isPinned) { _, pinned in
                    UserDefaults.standard.set(pinned, forKey: "windowPinnedEnabled")
                    applyPin(pinned)
                }
        }
        .windowResizability(.automatic)

        MenuBarExtra("Cursor 用量", systemImage: "gauge.with.dots.needle.67percent") {
            Button("粘贴 Cookie") { store.showCookieSheet = true }
            Button(store.isRefreshing ? "正在刷新…" : "立即刷新") {
                Task { await store.refresh() }
            }
            .disabled(store.isRefreshing)
            Toggle("置顶", isOn: $store.isPinned)
            if store.hasSession {
                Button("退出登录") { store.logout() }
            }
            Divider()
            Button("退出") { NSApplication.shared.terminate(nil) }
        }
    }

    private func applyPin(_ pinned: Bool) {
        WindowPinProbe.apply(pinned)
    }
}

struct WindowPinProbe: NSViewRepresentable {
    var pinned: Bool

    func makeNSView(context: Context) -> NSView {
        let view = NSView(frame: .zero)
        DispatchQueue.main.async { Self.apply(pinned, window: view.window) }
        return view
    }

    func updateNSView(_ view: NSView, context: Context) {
        Self.apply(pinned, window: view.window)
    }

    static func apply(_ pinned: Bool, window: NSWindow? = nil) {
        let windows = window.map { [$0] } ?? NSApplication.shared.windows
        for item in windows where item.level != .statusBar {
            item.level = pinned ? .floating : .normal
            item.hidesOnDeactivate = false
            item.isMovableByWindowBackground = true
            item.collectionBehavior.insert(.moveToActiveSpace)
        }
    }
}

struct DashboardView: View {
    @ObservedObject var store: UsageStore
    @State private var selectedPage: DashboardPage = .overview

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
                .padding(.horizontal, pagePadding)
                .padding(.top, 18)
                .padding(.bottom, 12)
            HStack(spacing: 8) {
                ForEach(DashboardPage.allCases) { page in
                    let selected = selectedPage == page
                    Button {
                        selectedPage = page
                    } label: {
                        Text(page.rawValue)
                            .font(.subheadline.weight(selected ? .semibold : .medium))
                            .foregroundStyle(selected ? .white : Color.white.opacity(0.78))
                            .padding(.horizontal, 14)
                            .padding(.vertical, 6)
                            .background(
                                Capsule().fill(selected ? Color.cyan.opacity(0.38) : Color.white.opacity(0.08))
                            )
                    }
                    .buttonStyle(.plain)
                }
                Spacer(minLength: 0)
            }
            .padding(.horizontal, pagePadding)
            .padding(.bottom, 12)

            Group {
                switch selectedPage {
                case .overview:
                    OverviewPage(snapshot: store.snapshot, pricingLabel: store.pricingLabel)
                case .models:
                    ModelsPage(snapshot: store.snapshot)
                case .settings:
                    SettingsPage(store: store)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            Text(store.refreshStatus)
                .font(.caption)
                .foregroundStyle(.white.opacity(0.5))
                .padding(.horizontal, pagePadding)
                .padding(.vertical, 12)
        }
        .foregroundStyle(.white)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .sheet(isPresented: $store.showCookieSheet) {
            CookieSheet(store: store)
        }
    }

    private var header: some View {
        HStack(alignment: .center) {
            VStack(alignment: .leading, spacing: 3) {
                Text("Cursor 用量").font(.title3.bold())
                Text(headerSubtitle)
                    .font(.caption2)
                    .foregroundStyle(.white.opacity(0.5))
            }
            Spacer()
            Toggle("置顶", isOn: $store.isPinned)
                .toggleStyle(.switch)
                .controlSize(.small)
                .labelsHidden()
                .help(store.isPinned ? "已置顶，点击取消" : "点击后始终浮在最前")
            Text(store.isPinned ? "置顶" : "普通")
                .font(.caption2)
                .foregroundStyle(.white.opacity(0.55))
            Button("Cookie") { selectedPage = .settings }
                .buttonStyle(.bordered)
            Button(store.isRefreshing ? "刷新中" : "刷新") {
                Task { await store.refresh() }
            }
            .disabled(store.isRefreshing)
            .buttonStyle(.borderedProminent)
        }
    }

    private var headerSubtitle: String {
        guard store.snapshot.hasRealData, store.snapshot.updatedAt != .distantPast else {
            return "仅保存在这台 Mac 上"
        }
        let time = store.snapshot.updatedAt.formatted(date: .omitted, time: .shortened)
        return "仅本机保存 · 已更新 \(time)"
    }
}

private struct PageScroll<Content: View>: View {
    @ViewBuilder var content: (Bool) -> Content

    var body: some View {
        GeometryReader { geo in
            ScrollView {
                content(geo.size.width >= wideBreakpoint)
                    .padding(.horizontal, pagePadding)
                    .padding(.bottom, 8)
                    .frame(maxWidth: .infinity, alignment: .topLeading)
            }
        }
    }
}

struct OverviewPage: View {
    let snapshot: UsageSnapshot
    var pricingLabel: String = ""

    var body: some View {
        PageScroll { wide in
            VStack(alignment: .leading, spacing: 14) {
                Group {
                    if wide {
                        HStack(alignment: .top, spacing: 14) {
                            RemainingCard(snapshot: snapshot)
                                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
                            tokenMetrics
                                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
                        }
                    } else {
                        VStack(alignment: .leading, spacing: 14) {
                            RemainingCard(snapshot: snapshot)
                            tokenMetrics
                        }
                    }
                }
                Text("预估按官方标价，套餐内可能 $0。\(pricingLabel)")
                    .font(.caption2)
                    .foregroundStyle(.white.opacity(0.4))
                todayTokenCard
                QuotaCard(snapshot: snapshot)
            }
            .frame(maxWidth: .infinity, alignment: .topLeading)
        }
    }

    private var tokenMetrics: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Metric(title: "总计", value: snapshot.hasRealData ? formatTokens(snapshot.totalTokens) : "—")
                Metric(title: "套餐内", value: snapshot.hasRealData ? formatTokens(snapshot.includedTokens) : "—")
                Metric(title: "按需", value: snapshot.hasRealData ? formatTokens(snapshot.onDemandTokens) : "—")
            }
            HStack(spacing: 8) {
                Metric(
                    title: "Cursor 消耗",
                    value: snapshot.hasRealData ? formatEstimate(snapshot.sourceUSD(.cursor)) : "—",
                    subtitle: snapshot.hasRealData ? formatTokens(snapshot.cursorSourceTokens) : nil
                )
                Metric(
                    title: "其他模型消耗",
                    value: snapshot.hasRealData ? formatEstimate(snapshot.sourceUSD(.other)) : "—",
                    subtitle: snapshot.hasRealData ? formatTokens(snapshot.otherSourceTokens) : nil
                )
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private var todayTokenCard: some View {
        let today = snapshot.todayTokens
        return VStack(alignment: .leading, spacing: 10) {
            VStack(alignment: .leading, spacing: 4) {
                Text("今日 token 消耗")
                    .font(.caption)
                    .foregroundStyle(.white.opacity(0.55))
                Text(snapshot.hasRealData ? formatTokens(today.total) : "—")
                    .font(.title2.weight(.semibold).monospacedDigit())
            }
            HStack(spacing: 8) {
                Metric(title: "输入", value: snapshot.hasRealData ? formatTokens(today.input) : "—")
                Metric(title: "输出", value: snapshot.hasRealData ? formatTokens(today.output) : "—")
                Metric(title: "缓存读", value: snapshot.hasRealData ? formatTokens(today.cacheRead) : "—")
                Metric(title: "缓存写", value: snapshot.hasRealData ? formatTokens(today.cacheWrite) : "—")
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }
}

struct ModelsPage: View {
    let snapshot: UsageSnapshot

    var body: some View {
        PageScroll { wide in
            VStack(alignment: .leading, spacing: 14) {
                sourceRow
                Text("预估按官方标价，套餐内可能 $0；接口不返回发起程序。")
                    .font(.caption2)
                    .foregroundStyle(.white.opacity(0.4))
                if snapshot.models.isEmpty {
                    Text(snapshot.hasRealData ? "这个账期还没有按模型拆开的 token" : "粘贴 Cookie 后显示各模型消耗")
                        .font(.footnote)
                        .foregroundStyle(.white.opacity(0.5))
                        .padding(16)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                } else {
                    weatherRow
                    if wide {
                        HStack(alignment: .top, spacing: 14) {
                            modelTable(title: "Cursor 模型", rows: snapshot.cursorModels, tint: cursorTint)
                            modelTable(title: "其他模型", rows: snapshot.otherModels, tint: otherTint)
                        }
                    } else {
                        modelTable(title: "Cursor 模型", rows: snapshot.cursorModels, tint: cursorTint)
                        modelTable(title: "其他模型", rows: snapshot.otherModels, tint: otherTint)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .topLeading)
        }
    }

    private var sourceRow: some View {
        HStack(spacing: 8) {
            Metric(
                title: "Cursor",
                value: formatWan(snapshot.cursorSourceTokens),
                subtitle: formatEstimate(snapshot.sourceUSD(.cursor))
            )
            Metric(
                title: "Grok 机器人",
                value: formatWan(snapshot.grokBotSourceTokens),
                subtitle: formatEstimate(snapshot.sourceUSD(.grokBot))
            )
            Metric(
                title: "其他模型",
                value: formatWan(snapshot.otherSourceTokens),
                subtitle: formatEstimate(snapshot.sourceUSD(.other))
            )
        }
    }

    private var weatherRow: some View {
        HStack(alignment: .top, spacing: 8) {
            ForEach(Array(snapshot.models.prefix(5))) { model in
                let tint = model.group == .cursor ? cursorTint : otherTint
                VStack(spacing: 6) {
                    Circle().fill(tint).frame(width: 6, height: 6)
                    Text(displayModelName(model.name))
                        .font(.caption2.weight(.medium))
                        .multilineTextAlignment(.center)
                        .lineLimit(2)
                        .minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity)
                    Text(formatWan(model.totalTokens))
                        .font(.caption.weight(.semibold).monospacedDigit())
                    Text(formatEstimate(model.estimatedUSD))
                        .font(.system(size: 10))
                        .foregroundStyle(.white.opacity(0.45))
                    ShareBar(value: model.totalTokens, total: maxToken, tint: tint)
                        .frame(height: 3)
                }
                .padding(.vertical, 8)
                .padding(.horizontal, 4)
                .frame(maxWidth: .infinity)
                .background(.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
        }
    }

    private func modelTable(title: String, rows: [ModelUsage], tint: Color) -> some View {
        let total = rows.reduce(0) { $0 + $1.totalTokens }
        let priced = rows.compactMap(\.estimatedUSD).reduce(0, +)
        let hasPrice = rows.contains { $0.estimatedUSD != nil }
        return VStack(alignment: .leading, spacing: 8) {
            HStack {
                Circle().fill(tint).frame(width: 6, height: 6)
                Text(title).font(.caption.weight(.semibold))
                Spacer()
                VStack(alignment: .trailing, spacing: 1) {
                    Text(formatWan(total)).font(.caption.monospacedDigit()).foregroundStyle(.white.opacity(0.55))
                    Text(hasPrice ? formatEstimate(priced) : "无标价")
                        .font(.system(size: 10))
                        .foregroundStyle(.white.opacity(0.4))
                }
            }
            if rows.isEmpty {
                Text("无").font(.caption2).foregroundStyle(.white.opacity(0.35))
            } else {
                ForEach(rows) { model in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack(alignment: .firstTextBaseline) {
                            Text(displayModelName(model.name))
                                .font(.caption)
                                .lineLimit(1)
                            Spacer(minLength: 8)
                            VStack(alignment: .trailing, spacing: 1) {
                                Text(formatWan(model.totalTokens))
                                    .font(.caption.monospacedDigit())
                                    .foregroundStyle(.white.opacity(0.75))
                                Text(formatEstimate(model.estimatedUSD))
                                    .font(.system(size: 10))
                                    .foregroundStyle(.white.opacity(0.4))
                            }
                        }
                        ShareBar(value: model.totalTokens, total: max(total, 1), tint: tint)
                            .frame(height: 4)
                    }
                }
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .topLeading)
        .background(.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private var maxToken: Double {
        snapshot.models.map(\.totalTokens).max() ?? 1
    }
}

struct SettingsPage: View {
    @ObservedObject var store: UsageStore
    @State private var pasted = ""

    var body: some View {
        PageScroll { _ in
            VStack(alignment: .leading, spacing: 16) {
                Text("本机登录").font(.headline)
                Text("只保存到本机钥匙串，不会写入用量缓存，也不会上传。可粘整段 Cookie，或只粘 WorkosCursorSessionToken 的值。")
                    .font(.footnote)
                    .foregroundStyle(.white.opacity(0.55))
                    .fixedSize(horizontal: false, vertical: true)
                TextEditor(text: $pasted)
                    .font(.system(.body, design: .monospaced))
                    .foregroundStyle(.white)
                    .frame(minHeight: 140)
                    .scrollContentBackground(.hidden)
                    .padding(8)
                    .background(.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
                    .frame(maxWidth: .infinity)
                HStack {
                    if store.hasSession {
                        Button("退出登录", role: .destructive) { store.logout() }
                    }
                    Spacer()
                    Button("保存并刷新") {
                        let value = pasted
                        pasted = ""
                        Task { await store.saveCookie(value) }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(pasted.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }

                Divider().overlay(.white.opacity(0.12))

                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("窗口置顶").font(.headline)
                        Text("打开后始终浮在其他应用前面。").font(.caption).foregroundStyle(.white.opacity(0.5))
                    }
                    Spacer()
                    Toggle("置顶", isOn: $store.isPinned)
                        .toggleStyle(.switch)
                        .labelsHidden()
                }

                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("开机与后台刷新").font(.headline)
                        Text("主窗口关掉也会每 3 分钟静默拉一次用量，不弹窗。").font(.caption).foregroundStyle(.white.opacity(0.5))
                    }
                    Spacer()
                    Toggle("后台刷新", isOn: Binding(
                        get: { store.backgroundRefreshEnabled },
                        set: { store.setBackgroundRefresh($0) }
                    ))
                    .toggleStyle(.switch)
                    .labelsHidden()
                }

                Text("用量缓存在本机 Application Support，不含 Cookie。预估美元按官方标价本地计算，不是账单。\(store.pricingLabel)。后台刷新不弹窗。")
                    .font(.caption)
                    .foregroundStyle(.white.opacity(0.4))
                    .fixedSize(horizontal: false, vertical: true)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

struct RemainingCard: View {
    let snapshot: UsageSnapshot

    var body: some View {
        HStack(spacing: 16) {
            UsageRing(
                percent: snapshot.hasRealData ? snapshot.remainingPercent : nil,
                title: "剩余",
                subtitle: "Cursor 模型",
                tint: cursorTint
            )
            UsageRing(
                percent: snapshot.premiumRemainingPercent,
                title: "剩余",
                subtitle: "高级模型",
                tint: otherTint
            )

            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    PlanBadge(title: snapshot.hasRealData ? snapshot.planLabel : "未连接", connected: snapshot.hasRealData)
                    if snapshot.hasRealData, let days = snapshot.daysRemaining {
                        Text("还剩 \(days) 天")
                            .font(.caption)
                            .foregroundStyle(.white.opacity(0.65))
                    }
                }
                if snapshot.hasRealData {
                    Text(snapshot.cycleEndLabel.isEmpty ? "本月账期" : "账期到 \(snapshot.cycleEndLabel)")
                        .font(.title3.weight(.semibold))
                    Text("左对齐 Cursor 模型，右对齐收费的其他模型")
                        .font(.caption)
                        .foregroundStyle(.white.opacity(0.65))
                } else {
                    Text("还没有用量数据").font(.title3)
                    Text("到设置页粘贴 Cookie 后即可在本机刷新").font(.caption).foregroundStyle(.white.opacity(0.6))
                }
            }
            Spacer(minLength: 0)
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(cardBackground, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
    }
}

struct UsageRing: View {
    let percent: Int?
    let title: String
    let subtitle: String
    let tint: Color

    var body: some View {
        ZStack {
            Circle().stroke(.white.opacity(0.14), lineWidth: 9)
            Circle()
                .trim(from: 0, to: CGFloat(percent ?? 0) / 100)
                .stroke(tint, style: StrokeStyle(lineWidth: 9, lineCap: .round))
                .rotationEffect(.degrees(-90))
            VStack(spacing: 2) {
                Text(percent.map { "\($0)%" } ?? "—")
                    .font(.system(size: 28, weight: .medium, design: .rounded))
                    .monospacedDigit()
                    .minimumScaleFactor(0.7)
                Text(title).font(.caption2).foregroundStyle(.white.opacity(0.7))
                Text(subtitle).font(.system(size: 9)).foregroundStyle(.white.opacity(0.45))
            }
        }
        .frame(width: 104, height: 104)
    }
}

struct QuotaCard: View {
    let snapshot: UsageSnapshot

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            quotaRow(title: "Cursor 模型", percent: snapshot.cursorModelsPercent, tint: cursorTint, footnote: nil)
            quotaRow(
                title: "其他模型",
                percent: snapshot.otherModelsPercent,
                tint: otherTint,
                footnote: snapshot.monthlyLimit > 0 ? "含至少 $\(Int(snapshot.monthlyLimit)) API" : nil
            )
            quotaRow(
                title: "Grok 机器人",
                percent: snapshot.grokWeeklyPercent,
                tint: Color.orange.opacity(0.9),
                footnote: snapshot.grokFootnote
            )
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private func quotaRow(title: String, percent: Int?, tint: Color, footnote: String?) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(title).font(.subheadline.weight(.medium))
                if let footnote {
                    Text(footnote).font(.caption2).foregroundStyle(.white.opacity(0.4))
                }
                Spacer()
                Text(percent.map { "已用 \($0)%" } ?? "未返回")
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.white.opacity(0.6))
            }
            GeometryReader { proxy in
                ZStack(alignment: .leading) {
                    Capsule().fill(.white.opacity(0.1))
                    Capsule()
                        .fill(tint)
                        .frame(width: proxy.size.width * CGFloat(min(max(percent ?? 0, 0), 100)) / 100)
                }
            }
            .frame(height: 8)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct ShareBar: View {
    let value: Double
    let total: Double
    let tint: Color

    var body: some View {
        GeometryReader { proxy in
            ZStack(alignment: .leading) {
                Capsule().fill(.white.opacity(0.1))
                Capsule()
                    .fill(tint)
                    .frame(width: proxy.size.width * CGFloat(total <= 0 ? 0 : min(value / total, 1)))
            }
        }
    }
}

struct PlanBadge: View {
    let title: String
    var connected: Bool = true

    var body: some View {
        Text(title)
            .font(.caption.weight(.bold))
            .foregroundStyle(.white)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(connected ? cursorTint : Color.white.opacity(0.18), in: Capsule())
            .fixedSize()
    }
}

struct Metric: View {
    let title: String
    let value: String
    var subtitle: String? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title).font(.caption2).foregroundStyle(.white.opacity(0.5))
            Text(value).font(.subheadline.weight(.semibold).monospacedDigit())
            if let subtitle {
                Text(subtitle).font(.system(size: 10)).foregroundStyle(.white.opacity(0.4))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(.white.opacity(0.06), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}

struct CookieSheet: View {
    @ObservedObject var store: UsageStore
    @State private var pasted = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("粘贴 Cookie").font(.title2.bold())
            Text("只保存到本机钥匙串，不会写入用量缓存，也不会上传。可粘整段 Cookie，或只粘 WorkosCursorSessionToken 的值。")
                .font(.footnote)
                .foregroundStyle(.secondary)
            TextEditor(text: $pasted)
                .font(.system(.body, design: .monospaced))
                .frame(minHeight: 140)
                .scrollContentBackground(.hidden)
                .padding(8)
                .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 10))
            HStack {
                Button("取消") { store.showCookieSheet = false }
                Spacer()
                if store.hasSession {
                    Button("退出登录", role: .destructive) { store.logout() }
                }
                Button("保存并刷新") {
                    let value = pasted
                    pasted = ""
                    Task { await store.saveCookie(value) }
                }
                .buttonStyle(.borderedProminent)
                .disabled(pasted.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24)
        .frame(minWidth: 520, idealWidth: 640)
    }
}
