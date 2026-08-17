import Foundation
import SwiftUI
import WidgetKit

@MainActor
final class UsageStore: ObservableObject {
    @Published var snapshot: UsageSnapshot
    @Published var refreshStatus = "等待粘贴 Cookie"
    @Published var needsReauth = false
    @Published var hasSession = false
    @Published var showCookieSheet = false
    @Published var isPinned = false
    @Published var isRefreshing = false
    @Published var pricingLabel = PricingStore.syncLabel
    @Published var backgroundRefreshEnabled = LaunchAgent.isEnabled

    private let defaults = UserDefaults.standard
    private var timer: Timer?
    private var refreshGeneration = 0
    private var activeUserRefresh = 0
    private var eventTask: Task<Void, Never>?

    init() {
        isPinned = UserDefaults.standard.bool(forKey: "windowPinnedEnabled")
        snapshot = SnapshotStore.load(defaults: defaults)
        if snapshot.hasRealData {
            SnapshotStore.save(snapshot, defaults: defaults)
            WidgetCenter.shared.reloadAllTimelines()
        }
        hasSession = SessionKeychain.hasSession
        if hasSession {
            refreshStatus = snapshot.hasRealData ? "已加载本地缓存" : "正在读取用量"
            Task { await refresh() }
        } else if snapshot.hasRealData {
            refreshStatus = "已显示上次缓存 · 请重新粘贴 Cookie 后刷新"
            needsReauth = true
        }
        timer = Timer.scheduledTimer(withTimeInterval: 180, repeats: true) { [weak self] _ in
            Task { @MainActor in
                await self?.refresh(silently: true)
            }
        }
        LaunchAgent.apply()
    }

    func setBackgroundRefresh(_ enabled: Bool) {
        backgroundRefreshEnabled = enabled
        LaunchAgent.isEnabled = enabled
        LaunchAgent.apply()
    }

    func saveCookie(_ pasted: String) async {
        do {
            try SessionKeychain.save(pasted)
            hasSession = true
            needsReauth = false
            showCookieSheet = false
            refreshStatus = "已保存到本机钥匙串，正在读取用量"
            await refresh()
        } catch {
            refreshStatus = error.localizedDescription
        }
    }

    func logout() {
        SessionKeychain.delete()
        hasSession = false
        needsReauth = true
        refreshStatus = "已退出登录。上次用量仍留在本机。"
    }

    func refresh(silently: Bool = false) async {
        guard hasSession || SessionKeychain.hasSession else {
            hasSession = false
            if !silently {
                needsReauth = true
                refreshStatus = "还没有登录态。请先粘贴 Cookie。"
            }
            return
        }
        hasSession = true
        refreshGeneration += 1
        let generation = refreshGeneration
        eventTask?.cancel()
        if !silently {
            activeUserRefresh = generation
            isRefreshing = true
        }
        if PricingStore.needsSync() {
            Task { [weak self] in
                let changed = await PricingClient.syncIfNeeded()
                await MainActor.run {
                    self?.pricingLabel = PricingStore.syncLabel
                    if changed {
                        self?.objectWillChange.send()
                        WidgetCenter.shared.reloadAllTimelines()
                    }
                }
            }
        }
        do {
            let overview = try await DashboardClient.fetchOverview()
            guard generation == refreshGeneration else {
                finishUserRefresh(generation)
                return
            }
            apply(overview, status: "已更新概览 · \(shortTime(overview.updatedAt))")
            finishUserRefresh(generation)
            eventTask = Task { [weak self] in
                await self?.applyEventPatch(base: overview, generation: generation)
            }
        } catch DashboardError.unauthorized {
            guard generation == refreshGeneration else {
                finishUserRefresh(generation)
                return
            }
            SessionKeychain.delete()
            hasSession = false
            needsReauth = true
            refreshStatus = DashboardError.unauthorized.localizedDescription
            finishUserRefresh(generation)
        } catch {
            guard generation == refreshGeneration else {
                finishUserRefresh(generation)
                return
            }
            finishUserRefresh(generation)
            if !silently || !snapshot.hasRealData {
                refreshStatus = error.localizedDescription
            } else {
                refreshStatus = "仍显示缓存 · \(error.localizedDescription)"
            }
        }
    }

    private func finishUserRefresh(_ generation: Int) {
        if activeUserRefresh == generation {
            isRefreshing = false
        }
    }

    private func applyEventPatch(base: UsageSnapshot, generation: Int) async {
        do {
            let patched = try await DashboardClient.fetchEventPatch(for: base)
            guard !Task.isCancelled, generation == refreshGeneration else { return }
            apply(patched, status: "已更新 · \(shortTime(patched.updatedAt))")
        } catch {
            guard !Task.isCancelled, generation == refreshGeneration else { return }
        }
    }

    private func apply(_ value: UsageSnapshot, status: String) {
        snapshot = value
        SnapshotStore.save(value, defaults: defaults)
        WidgetCenter.shared.reloadAllTimelines()
        needsReauth = false
        refreshStatus = status
    }

    private func shortTime(_ date: Date) -> String {
        date.formatted(date: .omitted, time: .shortened)
    }
}

enum LaunchAgent {
    static let label = "com.local.cursorusage.refresh"
    private static let enabledKey = "backgroundRefreshEnabled"

    static var plistURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/LaunchAgents/\(label).plist")
    }

    static var installedExecutable: URL {
        if let url = Bundle.main.executableURL {
            return url
        }
        return FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Applications/Cursor用量.app/Contents/MacOS/CursorUsageWidget")
    }

    static var isEnabled: Bool {
        get {
            if UserDefaults.standard.object(forKey: enabledKey) == nil { return true }
            return UserDefaults.standard.bool(forKey: enabledKey)
        }
        set { UserDefaults.standard.set(newValue, forKey: enabledKey) }
    }

    static func apply() {
        if isEnabled { install() } else { uninstall() }
    }

    static func install() {
        let folder = plistURL.deletingLastPathComponent()
        try? FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        let plist: [String: Any] = [
            "Label": label,
            "RunAtLoad": true,
            "StartInterval": 180,
            "ProgramArguments": [installedExecutable.path, "--refresh"],
            "StandardOutPath": "/dev/null",
            "StandardErrorPath": "/dev/null"
        ]
        guard let data = try? PropertyListSerialization.data(fromPropertyList: plist, format: .xml, options: 0) else { return }
        try? data.write(to: plistURL, options: .atomic)
        let domain = "gui/\(getuid())"
        _ = run("/bin/launchctl", ["bootout", "\(domain)/\(label)"])
        _ = run("/bin/launchctl", ["bootstrap", domain, plistURL.path])
    }

    static func uninstall() {
        let domain = "gui/\(getuid())"
        _ = run("/bin/launchctl", ["bootout", "\(domain)/\(label)"])
        try? FileManager.default.removeItem(at: plistURL)
    }

    @discardableResult
    private static func run(_ launchPath: String, _ arguments: [String]) -> Int32 {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: launchPath)
        process.arguments = arguments
        process.standardOutput = FileHandle.nullDevice
        process.standardError = FileHandle.nullDevice
        try? process.run()
        process.waitUntilExit()
        return process.terminationStatus
    }
}

enum BackgroundRefresh {
    static func runAndExitIfNeeded() {
        guard CommandLine.arguments.contains("--refresh") else { return }
        let loop = CFRunLoopGetCurrent()
        Task.detached {
            await runOnce()
            CFRunLoopStop(loop)
        }
        CFRunLoopRun()
        exit(0)
    }

    static func runOnce() async {
        let priceChanged = await PricingClient.syncIfNeeded()
        guard SessionKeychain.hasSession else { return }
        do {
            let overview = try await DashboardClient.fetchOverview()
            let patched = (try? await DashboardClient.fetchEventPatch(for: overview)) ?? overview
            SnapshotStore.save(patched)
            WidgetCenter.shared.reloadAllTimelines()
        } catch {
            if priceChanged {
                WidgetCenter.shared.reloadAllTimelines()
            }
        }
    }
}
