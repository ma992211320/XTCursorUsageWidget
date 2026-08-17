import Foundation

enum DashboardError: LocalizedError {
    case noSession
    case unauthorized
    case empty
    case requestFailed(String)

    var errorDescription: String? {
        switch self {
        case .noSession: return "还没有登录态。请先粘贴 Cookie。"
        case .unauthorized: return "Cookie 已失效，请重新粘贴"
        case .empty: return "接口没有返回用量字段"
        case .requestFailed(let path): return "读取 \(path) 失败"
        }
    }
}

enum DashboardClient {
    private static let origin = "https://cursor.com"
    private static let sharedSession: URLSession = {
        let config = URLSessionConfiguration.ephemeral
        config.httpCookieStorage = nil
        config.httpShouldSetCookies = false
        config.timeoutIntervalForRequest = 12
        config.httpMaximumConnectionsPerHost = 6
        return URLSession(configuration: config)
    }()
    private static var lastUserId: Double?
    private static var eventsUseTeamId = true
    private static var preferredAggregated = AggregatedBody.team0

    static func fetchOverview() async throws -> UsageSnapshot {
        guard SessionKeychain.read() != nil else { throw DashboardError.noSession }

        async let meTask = get("/api/auth/me")
        async let periodTask = optionalPost("/api/dashboard/get-current-period-usage", body: [:])
        async let planTask = optionalPost("/api/dashboard/get-plan-info", body: [:])
        async let sandTask = optionalPost("/api/dashboard/get-sand-usage-status", body: [:])

        let me = try await meTask
        guard let userId = UsageJSON.number(me["id"]) else { throw DashboardError.unauthorized }
        lastUserId = userId

        let now = Date().timeIntervalSince1970 * 1000
        let monthStart = startOfLocalMonth()
        async let aggregatedTask = fetchAggregated(userId: userId, startMs: monthStart, endMs: now)

        let period = try await periodTask
        let plan = try await planTask
        let sand = try await sandTask

        var summary: [String: Any]?
        if period == nil {
            summary = try await optionalGet("/api/usage-summary")
        }

        let startMs = UsageJSON.toMs(period?["billingCycleStart"])
            ?? UsageJSON.toMs(summary?["billingCycleStart"])
            ?? monthStart
        let endMs = max(
            UsageJSON.toMs(period?["billingCycleEnd"]) ?? UsageJSON.toMs(summary?["billingCycleEnd"]) ?? now,
            now
        )

        var aggregated = try await aggregatedTask
        if abs(startMs - monthStart) > 12 * 60 * 60 * 1000 {
            aggregated = try await fetchAggregated(userId: userId, startMs: startMs, endMs: endMs) ?? aggregated
        }
        guard let snapshot = UsageParser.build(
            period: period,
            summary: summary,
            plan: plan,
            aggregated: aggregated,
            todayEvents: [],
            cycleEvents: [],
            sand: sand,
            now: Date()
        ) else { throw DashboardError.empty }
        return snapshot
    }

    static func fetchEventPatch(for snapshot: UsageSnapshot) async throws -> UsageSnapshot {
        guard SessionKeychain.read() != nil else { throw DashboardError.noSession }
        let userId: Double
        if let cached = lastUserId {
            userId = cached
        } else if let fetched = UsageJSON.number((try await get("/api/auth/me"))["id"]) {
            userId = fetched
        } else {
            throw DashboardError.unauthorized
        }
        lastUserId = userId

        let now = Date().timeIntervalSince1970 * 1000
        let startMs = UsageJSON.toMs(snapshot.billingCycleStart) ?? startOfLocalMonth()
        let endMs = max(UsageJSON.toMs(snapshot.billingCycleEnd) ?? now, now)
        let todayStart = startOfLocalDay()
        let sameWindow = abs(startMs - todayStart) < 12 * 60 * 60 * 1000

        let cycleEvents = try await fetchEvents(userId: userId, startMs: startMs, endMs: endMs, maxPages: 2)
        let todayEvents = sameWindow
            ? cycleEvents
            : (try await fetchEvents(userId: userId, startMs: todayStart, endMs: now, maxPages: 1))
        return UsageParser.patchEvents(snapshot, todayEvents: todayEvents, cycleEvents: cycleEvents)
    }

    static func fetchSnapshot() async throws -> UsageSnapshot {
        let overview = try await fetchOverview()
        return (try? await fetchEventPatch(for: overview)) ?? overview
    }

    private static func request(path: String, method: String, body: [String: Any]?) throws -> URLRequest {
        guard let token = SessionKeychain.read() else { throw DashboardError.noSession }
        guard let url = URL(string: origin + path) else { throw DashboardError.requestFailed(path) }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue(origin, forHTTPHeaderField: "Origin")
        request.setValue(origin + "/dashboard/usage", forHTTPHeaderField: "Referer")
        request.setValue("WorkosCursorSessionToken=\(token)", forHTTPHeaderField: "Cookie")
        if let body {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: body)
        }
        return request
    }

    private static func send(_ request: URLRequest, path: String) async throws -> [String: Any] {
        let (data, response) = try await sharedSession.data(for: request)
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0
        if status == 204 || status == 401 || status == 403 { throw DashboardError.unauthorized }
        guard (200...299).contains(status) else { throw DashboardError.requestFailed(path) }
        guard !data.isEmpty,
              let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw DashboardError.requestFailed(path)
        }
        return object
    }

    private static func get(_ path: String) async throws -> [String: Any] {
        try await send(try request(path: path, method: "GET", body: nil), path: path)
    }

    private static func post(_ path: String, body: [String: Any]) async throws -> [String: Any] {
        try await send(try request(path: path, method: "POST", body: body), path: path)
    }

    private static func optionalGet(_ path: String) async throws -> [String: Any]? {
        do { return try await get(path) }
        catch let error as DashboardError where error.isUnauthorized { throw error }
        catch { return nil }
    }

    private static func optionalPost(_ path: String, body: [String: Any]) async throws -> [String: Any]? {
        do { return try await post(path, body: body) }
        catch let error as DashboardError where error.isUnauthorized { throw error }
        catch { return nil }
    }

    private enum AggregatedBody: Int, CaseIterable {
        case team0
        case teamNeg1
        case datesOnly

        func payload(userId: Double, startMs: Double, endMs: Double) -> [String: Any] {
            let start = String(Int(startMs))
            let end = String(Int(endMs))
            switch self {
            case .team0: return ["teamId": 0, "userId": userId, "startDate": start, "endDate": end]
            case .teamNeg1: return ["teamId": -1, "userId": userId, "startDate": start, "endDate": end]
            case .datesOnly: return ["startDate": start, "endDate": end]
            }
        }
    }

    private static func fetchAggregated(userId: Double, startMs: Double, endMs: Double) async throws -> [String: Any]? {
        var kinds = AggregatedBody.allCases
        if let index = kinds.firstIndex(of: preferredAggregated), index > 0 {
            kinds.move(fromOffsets: IndexSet(integer: index), toOffset: 0)
        }
        for kind in kinds {
            do {
                let data = try await post(
                    "/api/dashboard/get-aggregated-usage-events",
                    body: kind.payload(userId: userId, startMs: startMs, endMs: endMs)
                )
                if data["aggregations"] != nil || data["totalInputTokens"] != nil || data["totalOutputTokens"] != nil {
                    preferredAggregated = kind
                    return data
                }
            } catch let error as DashboardError where error.isUnauthorized {
                throw error
            } catch {
                continue
            }
        }
        return nil
    }

    private static func fetchEvents(userId: Double, startMs: Double, endMs: Double, maxPages: Int) async throws -> [[String: Any]] {
        var events: [[String: Any]] = []
        var page = 1
        var total = Int.max
        while events.count < total && page <= maxPages {
            let data = try await fetchEventPage(userId: userId, startMs: startMs, endMs: endMs, page: page)
            let batch = UsageJSON.array(data?["usageEventsDisplay"]) ?? UsageJSON.array(data?["usageEvents"]) ?? []
            total = UsageJSON.int(data?["totalUsageEventsCount"]) ?? (events.count + batch.count)
            let rows = batch.compactMap { UsageJSON.object($0) }
            if rows.isEmpty { break }
            events.append(contentsOf: rows)
            page += 1
        }
        return events
    }

    private static func fetchEventPage(userId: Double, startMs: Double, endMs: Double, page: Int) async throws -> [String: Any]? {
        let withTeam: [String: Any] = [
            "teamId": 0, "userId": userId, "startDate": String(Int(startMs)),
            "endDate": String(Int(endMs)), "page": page, "pageSize": 200
        ]
        let withoutTeam: [String: Any] = [
            "userId": userId, "startDate": String(Int(startMs)),
            "endDate": String(Int(endMs)), "page": page, "pageSize": 200
        ]
        let first = eventsUseTeamId ? withTeam : withoutTeam
        let second = eventsUseTeamId ? withoutTeam : withTeam
        do {
            return try await post("/api/dashboard/get-filtered-usage-events", body: first)
        } catch let error as DashboardError where error.isUnauthorized {
            throw error
        } catch {
            eventsUseTeamId.toggle()
            do {
                return try await post("/api/dashboard/get-filtered-usage-events", body: second)
            } catch let error as DashboardError where error.isUnauthorized {
                throw error
            } catch {
                return nil
            }
        }
    }

    private static func startOfLocalDay() -> Double {
        Calendar.current.startOfDay(for: Date()).timeIntervalSince1970 * 1000
    }

    private static func startOfLocalMonth() -> Double {
        let now = Date()
        let parts = Calendar.current.dateComponents([.year, .month], from: now)
        return (Calendar.current.date(from: parts) ?? now).timeIntervalSince1970 * 1000
    }
}

private extension DashboardError {
    var isUnauthorized: Bool {
        if case .unauthorized = self { return true }
        return false
    }
}

enum UsageParser {
    static func build(
        period: [String: Any]?,
        summary: [String: Any]?,
        plan: [String: Any]?,
        aggregated: [String: Any]?,
        todayEvents: [[String: Any]],
        cycleEvents: [[String: Any]],
        sand: [String: Any]? = nil,
        now: Date
    ) -> UsageSnapshot? {
        if period == nil && summary == nil && aggregated == nil { return nil }

        let planUsage = UsageJSON.object(period?["planUsage"])
            ?? UsageJSON.object(UsageJSON.object(summary?["individualUsage"])?["plan"])
            ?? [:]
        let planInfo = UsageJSON.object(plan?["planInfo"]) ?? [:]
        let includedSpendCents = UsageJSON.number(planUsage["includedSpend"])
            ?? UsageJSON.number(planUsage["used"])
            ?? UsageJSON.number(planUsage["totalSpend"])
            ?? 0
        let limitCents = UsageJSON.number(planUsage["limit"])
            ?? UsageJSON.number(planInfo["includedAmountCents"])
            ?? 0
        let displayMessage = UsageJSON.string(period?["displayMessage"])
            ?? UsageJSON.string(summary?["displayMessage"])
            ?? ""
        var includedPercent = 0
        if limitCents > 0 {
            includedPercent = Int(((includedSpendCents / limitCents) * 100).rounded())
        } else {
            includedPercent = UsageJSON.percent(from: displayMessage) ?? 0
        }

        let startMs = UsageJSON.toMs(period?["billingCycleStart"]) ?? UsageJSON.toMs(summary?["billingCycleStart"])
        let endMs = UsageJSON.toMs(period?["billingCycleEnd"]) ?? UsageJSON.toMs(summary?["billingCycleEnd"])
        let tokens = tokensFromAggregated(aggregated)
        let models = modelsFromAggregated(aggregated)
        let todayTokens = todayEvents.reduce(TokenTotals.zero) { $0 + UsageJSON.tokens(from: $1) }
        let split = splitTokens(cycleEvents)
        var grokRoot: [String: Any] = [:]
        if let period { grokRoot.merge(period) { _, new in new } }
        if let summary { grokRoot.merge(summary) { _, new in new } }
        if let plan { grokRoot.merge(plan) { _, new in new } }
        var grok = grokFields(from: grokRoot)
        let sandGrok = grokFromSand(sand)
        if sandGrok.percent != nil { grok.percent = sandGrok.percent }
        if !sandGrok.reset.isEmpty { grok.reset = sandGrok.reset }
        if grok.reset.isEmpty {
            grok.reset = weeklyReset(from: startMs, now: now)
        }

        return UsageSnapshot(
            monthlyLimit: limitCents / 100,
            monthlyUsed: includedSpendCents / 100,
            todayUsed: 0,
            models: models,
            updatedAt: now,
            includedPercent: includedPercent,
            displayMessage: displayMessage,
            billingCycleStart: UsageJSON.iso(startMs),
            billingCycleEnd: UsageJSON.iso(endMs),
            planName: UsageJSON.string(planInfo["planName"])
                ?? UsageJSON.string(period?["membershipType"])
                ?? UsageJSON.string(summary?["membershipType"])
                ?? "",
            tokens: tokens,
            todayTokens: todayTokens,
            hasRealData: true,
            source: "dashboard-json",
            totalTokens: split.total > 0 ? split.total : tokens.total,
            includedTokens: split.total > 0 ? split.included : tokens.total,
            onDemandTokens: split.total > 0 ? split.onDemand : 0,
            cursorModelsPercent: UsageJSON.percent(from: UsageJSON.string(period?["autoModelSelectedDisplayMessage"]))
                ?? UsageJSON.int(planUsage["autoPercentUsed"]),
            otherModelsPercent: UsageJSON.percent(from: UsageJSON.string(period?["namedModelSelectedDisplayMessage"]))
                ?? UsageJSON.int(planUsage["apiPercentUsed"]),
            grokWeeklyPercent: grok.percent,
            grokResetAt: grok.reset,
            days: daysFromEvents(cycleEvents)
        )
    }

    static func tokensFromAggregated(_ data: [String: Any]?) -> TokenTotals {
        guard let data else { return .zero }
        var totals = TokenTotals(
            input: UsageJSON.number(data["totalInputTokens"]) ?? 0,
            output: UsageJSON.number(data["totalOutputTokens"]) ?? 0,
            cacheRead: UsageJSON.number(data["totalCacheReadTokens"]) ?? 0,
            cacheWrite: UsageJSON.number(data["totalCacheWriteTokens"]) ?? 0,
            total: 0
        )
        if totals.input + totals.output + totals.cacheRead + totals.cacheWrite == 0 {
            for row in UsageJSON.array(data["aggregations"]) ?? [] {
                totals = totals + UsageJSON.tokens(from: row)
            }
        }
        totals.total = totals.input + totals.output + totals.cacheRead + totals.cacheWrite
        return totals
    }

    static func modelsFromAggregated(_ data: [String: Any]?) -> [ModelUsage] {
        let rows = UsageJSON.array(data?["aggregations"]) ?? []
        var models = rows.compactMap { row -> ModelUsage? in
            guard let object = UsageJSON.object(row) else { return nil }
            let name = UsageJSON.string(object["modelIntent"]) ?? UsageJSON.string(object["model"]) ?? "unknown"
            let tokens = UsageJSON.tokens(from: object)
            guard tokens.total > 0 || (UsageJSON.number(object["totalCents"]) ?? 0) > 0 else { return nil }
            return ModelUsage(
                name: name,
                group: ModelUsage.group(for: name),
                tokens: tokens,
                percent: 0,
                amount: (UsageJSON.number(object["totalCents"]) ?? 0) / 100
            )
        }
        let cursorTotal = models.filter { $0.group == .cursor }.reduce(0) { $0 + $1.tokens.total }
        let otherTotal = models.filter { $0.group == .other }.reduce(0) { $0 + $1.tokens.total }
        models = models.map { model in
            var copy = model
            let groupTotal = model.group == .cursor ? cursorTotal : otherTotal
            copy.percent = groupTotal > 0 ? (model.tokens.total / groupTotal) * 100 : 0
            return copy
        }
        return models.sorted { $0.tokens.total > $1.tokens.total }
    }

    static func splitTokens(_ events: [[String: Any]]) -> (total: Double, included: Double, onDemand: Double) {
        var included = 0.0
        var onDemand = 0.0
        for event in events {
            let tokens = UsageJSON.tokens(from: event).total
            let kind = (UsageJSON.string(event["kind"]) ?? "").uppercased()
            if kind.contains("INCLUDED") {
                included += tokens
            } else if kind.contains("USAGE_BASED") || kind.contains("ON_DEMAND") || kind.contains("ONDEMAND") {
                onDemand += tokens
            } else {
                included += tokens
            }
        }
        return (included + onDemand, included, onDemand)
    }

    static func patchEvents(
        _ snapshot: UsageSnapshot,
        todayEvents: [[String: Any]],
        cycleEvents: [[String: Any]]
    ) -> UsageSnapshot {
        var next = snapshot
        let split = splitTokens(cycleEvents)
        if split.total > 0 {
            next.totalTokens = max(snapshot.totalTokens, split.total)
            next.includedTokens = split.included
            next.onDemandTokens = split.onDemand
        }
        next.todayTokens = todayEvents.reduce(TokenTotals.zero) { $0 + UsageJSON.tokens(from: $1) }
        next.days = daysFromEvents(cycleEvents)
        return next
    }

    static func daysFromEvents(_ events: [[String: Any]]) -> [DayUsage] {
        var buckets: [String: [String: Double]] = [:]
        for event in events {
            let timestamp = UsageJSON.toMs(event["timestamp"]) ?? UsageJSON.toMs(event["eventDate"])
            guard let timestamp else { continue }
            let day = UsageJSON.utcDay(Date(timeIntervalSince1970: timestamp / 1000))
            let name = UsageJSON.string(event["model"]) ?? UsageJSON.string(event["modelIntent"]) ?? "unknown"
            let tokens = UsageJSON.tokens(from: event).total
            buckets[day, default: [:]][name, default: 0] += tokens
        }
        return buckets.keys.sorted().map { DayUsage(date: $0, models: buckets[$0] ?? [:]) }
    }

    static func grokFromSand(_ sand: [String: Any]?) -> (percent: Int?, reset: String) {
        guard let sand else { return (nil, "") }
        let percent = UsageJSON.int(sand["usagePercent"]) ?? UsageJSON.int(sand["usage_percent"])
        let reset = UsageJSON.string(sand["nextResetTimestampUtc"])
            ?? UsageJSON.string(sand["next_reset_timestamp_utc"])
            ?? UsageJSON.iso(UsageJSON.toMs(sand["nextResetTimestampUtc"]) ?? UsageJSON.toMs(sand["next_reset_timestamp_utc"]))
        return (percent, reset)
    }

    static func grokFields(from root: [String: Any]) -> (percent: Int?, reset: String) {
        var percent: Int?
        var reset = ""
        walk(root) { key, value in
            let name = key.lowercased()
            let grokish = name.contains("grok")
            let weekly = name.contains("weekly") || name.contains("week")
            let botish = name.contains("bot") && (grokish || weekly || name.contains("grok"))
            if grokish || botish || (weekly && name.contains("percent") && name.contains("grok")) {
                if percent == nil {
                    percent = UsageJSON.int(value) ?? UsageJSON.percent(from: value as? String)
                }
            }
            if let object = UsageJSON.object(value), grokish || botish || weekly {
                if percent == nil {
                    percent = UsageJSON.int(object["percent"])
                        ?? UsageJSON.int(object["usedPercent"])
                        ?? UsageJSON.percent(from: UsageJSON.string(object["displayMessage"]))
                }
                if reset.isEmpty {
                    reset = UsageJSON.string(object["resetAt"])
                        ?? UsageJSON.string(object["resetsAt"])
                        ?? UsageJSON.string(object["resetDate"])
                        ?? UsageJSON.iso(UsageJSON.toMs(object["resetAt"]) ?? UsageJSON.toMs(object["resetsAt"]))
                }
            }
            if reset.isEmpty && (name.contains("grok") && name.contains("reset") || name == "weeklyreset") {
                reset = UsageJSON.string(value) ?? UsageJSON.iso(UsageJSON.toMs(value))
            }
        }
        return (percent, reset)
    }

    static func weeklyReset(from startMs: Double?, now: Date) -> String {
        guard let startMs else { return "" }
        var reset = Date(timeIntervalSince1970: startMs / 1000).addingTimeInterval(7 * 24 * 60 * 60)
        while reset <= now {
            reset = reset.addingTimeInterval(7 * 24 * 60 * 60)
        }
        return ISO8601DateFormatter().string(from: reset)
    }

    private static func walk(_ value: Any, visit: (String, Any) -> Void) {
        guard let object = UsageJSON.object(value) else {
            if let array = UsageJSON.array(value) {
                array.forEach { walk($0, visit: visit) }
            }
            return
        }
        for (key, child) in object {
            visit(key, child)
            walk(child, visit: visit)
        }
    }
}

enum PricingClient {
    private static let session: URLSession = {
        let config = URLSessionConfiguration.ephemeral
        config.httpCookieStorage = nil
        config.httpShouldSetCookies = false
        config.timeoutIntervalForRequest = 12
        return URLSession(configuration: config)
    }()

    @discardableResult
    static func syncIfNeeded() async -> Bool {
        guard PricingStore.needsSync() else { return false }
        PricingStore.markAttempted()
        guard let url = URL(string: "https://cursor.com/docs/models-and-pricing.md") else { return false }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.setValue("text/markdown,text/plain,*/*", forHTTPHeaderField: "Accept")
        do {
            let (data, response) = try await session.data(for: request)
            let status = (response as? HTTPURLResponse)?.statusCode ?? 0
            guard (200...299).contains(status),
                  let text = String(data: data, encoding: .utf8),
                  let catalog = PricingParser.parse(text) else { return false }
            return PricingStore.saveIfChanged(catalog)
        } catch {
            return false
        }
    }
}
