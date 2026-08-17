import Darwin
import Foundation

enum SnapshotStore {
    static var supportFolder: URL {
        realHomeDirectory().appendingPathComponent("Library/Application Support/CursorUsageWidget", isDirectory: true)
    }

    static var supportURL: URL {
        supportFolder.appendingPathComponent("snapshot.json")
    }

    static func load(defaults: UserDefaults? = .standard) -> UsageSnapshot {
        if let data = try? Data(contentsOf: supportURL), let value = decode(data), value.hasRealData {
            return value
        }
        if let data = defaults?.data(forKey: "usageSnapshot"), let value = decode(data), value.hasRealData {
            return value
        }
        return .empty
    }

    static func save(_ snapshot: UsageSnapshot, defaults: UserDefaults? = .standard) {
        guard let data = try? JSONEncoder().encode(snapshot) else { return }
        defaults?.set(data, forKey: "usageSnapshot")
        defaults?.synchronize()
        try? FileManager.default.createDirectory(at: supportFolder, withIntermediateDirectories: true)
        try? data.write(to: supportURL, options: .atomic)
    }

    static func storageFolders() -> [URL] {
        [supportFolder]
    }

    private static func decode(_ data: Data) -> UsageSnapshot? {
        try? JSONDecoder().decode(UsageSnapshot.self, from: data)
    }
}

func realHomeDirectory() -> URL {
    if let pw = getpwuid(getuid()), let dir = pw.pointee.pw_dir {
        return URL(fileURLWithPath: String(cString: dir), isDirectory: true)
    }
    return FileManager.default.homeDirectoryForCurrentUser
}

struct TokenTotals: Codable, Equatable {
    var input: Double
    var output: Double
    var cacheRead: Double
    var cacheWrite: Double
    var total: Double

    static let zero = TokenTotals(input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0)

    static func + (lhs: TokenTotals, rhs: TokenTotals) -> TokenTotals {
        TokenTotals(
            input: lhs.input + rhs.input,
            output: lhs.output + rhs.output,
            cacheRead: lhs.cacheRead + rhs.cacheRead,
            cacheWrite: lhs.cacheWrite + rhs.cacheWrite,
            total: lhs.total + rhs.total
        )
    }
}

enum ModelGroup: String, Codable {
    case cursor
    case other
}

enum TokenSource: String {
    case cursor
    case grokBot
    case other

    static func of(_ name: String) -> TokenSource {
        let key = name.lowercased()
        if key.hasPrefix("sand-") { return .grokBot }
        if key.hasPrefix("cursor-") || key.hasPrefix("composer-") || key == "auto" {
            return .cursor
        }
        return .other
    }

    var label: String {
        switch self {
        case .cursor: return "Cursor"
        case .grokBot: return "Grok 机器人"
        case .other: return "其他模型"
        }
    }
}

struct ModelUsage: Codable, Identifiable, Equatable {
    var name: String
    var group: ModelGroup
    var tokens: TokenTotals
    var percent: Double
    var amount: Double

    var id: String { name }
    var totalTokens: Double { tokens.total }
    var estimatedUSD: Double? { ModelPricing.estimate(name: name, tokens: tokens) }

    static func group(for name: String) -> ModelGroup {
        let key = name.lowercased()
        if key.hasPrefix("cursor-") || key.hasPrefix("composer-") || key == "auto" {
            return .cursor
        }
        return .other
    }
}

struct DayUsage: Codable, Identifiable, Equatable {
    var date: String
    var models: [String: Double]

    var id: String { date }
    var total: Double { models.values.reduce(0, +) }
}

struct UsageSnapshot: Codable, Equatable {
    var monthlyLimit: Double
    var monthlyUsed: Double
    var todayUsed: Double
    var models: [ModelUsage]
    var updatedAt: Date
    var includedPercent: Int
    var displayMessage: String
    var billingCycleStart: String
    var billingCycleEnd: String
    var planName: String
    var tokens: TokenTotals
    var todayTokens: TokenTotals
    var hasRealData: Bool
    var source: String
    var totalTokens: Double
    var includedTokens: Double
    var onDemandTokens: Double
    var cursorModelsPercent: Int?
    var otherModelsPercent: Int?
    var grokWeeklyPercent: Int?
    var grokResetAt: String
    var days: [DayUsage]

    static let empty = UsageSnapshot(
        monthlyLimit: 0,
        monthlyUsed: 0,
        todayUsed: 0,
        models: [],
        updatedAt: .distantPast,
        includedPercent: 0,
        displayMessage: "还没有用量数据",
        billingCycleStart: "",
        billingCycleEnd: "",
        planName: "",
        tokens: .zero,
        todayTokens: .zero,
        hasRealData: false,
        source: "",
        totalTokens: 0,
        includedTokens: 0,
        onDemandTokens: 0,
        cursorModelsPercent: nil,
        otherModelsPercent: nil,
        grokWeeklyPercent: nil,
        grokResetAt: "",
        days: []
    )

    static let sample = UsageSnapshot(
        monthlyLimit: 400,
        monthlyUsed: 12.88,
        todayUsed: 0,
        models: [],
        updatedAt: .now,
        includedPercent: 1,
        displayMessage: "",
        billingCycleStart: "",
        billingCycleEnd: "",
        planName: "Ultra",
        tokens: TokenTotals(input: 20_000_000, output: 1_000_000, cacheRead: 16_000_000, cacheWrite: 1_000_000, total: 38_000_000),
        todayTokens: TokenTotals(input: 8_000_000, output: 400_000, cacheRead: 6_000_000, cacheWrite: 400_000, total: 14_800_000),
        hasRealData: true,
        source: "sample",
        totalTokens: 38_000_000,
        includedTokens: 38_000_000,
        onDemandTokens: 0,
        cursorModelsPercent: 1,
        otherModelsPercent: 0,
        grokWeeklyPercent: 1,
        grokResetAt: "",
        days: []
    )

    var remaining: Double { max(0, monthlyLimit - monthlyUsed) }

    var remainingPercent: Int {
        if !hasRealData { return 0 }
        if let used = cursorModelsPercent {
            return max(0, min(100, 100 - used))
        }
        if monthlyLimit == 0 && includedPercent == 0 { return 0 }
        return max(0, min(100, 100 - includedPercent))
    }

    /// Spending「其他模型」剩余，也就是按 API 标价扣的收费池。
    var premiumRemainingPercent: Int? {
        guard hasRealData, let used = otherModelsPercent else { return nil }
        return max(0, min(100, 100 - used))
    }

    var planLabel: String {
        let name = planName.trimmingCharacters(in: .whitespacesAndNewlines)
        return name.isEmpty ? "个人" : name
    }

    var billingEndDate: Date? { UsageJSON.date(from: billingCycleEnd) }

    var daysRemaining: Int? {
        guard let end = billingEndDate else { return nil }
        let calendar = Calendar.current
        let start = calendar.startOfDay(for: Date())
        let finish = calendar.startOfDay(for: end)
        return max(0, calendar.dateComponents([.day], from: start, to: finish).day ?? 0)
    }

    var cycleEndLabel: String {
        guard let end = billingEndDate else { return "" }
        return UsageJSON.monthDay(end)
    }

    var grokResetLabel: String {
        guard let date = UsageJSON.date(from: grokResetAt) else { return grokResetAt }
        return UsageJSON.monthDay(date)
    }

    var grokDaysRemaining: Int? {
        guard let end = UsageJSON.date(from: grokResetAt) else { return nil }
        let calendar = Calendar.current
        let start = calendar.startOfDay(for: Date())
        let finish = calendar.startOfDay(for: end)
        return max(0, calendar.dateComponents([.day], from: start, to: finish).day ?? 0)
    }

    var grokFootnote: String {
        if grokResetLabel.isEmpty { return "周重置时间未返回" }
        if let days = grokDaysRemaining {
            return "重置 \(grokResetLabel)（还剩 \(days) 天）"
        }
        return "重置 \(grokResetLabel)"
    }

    var cursorModels: [ModelUsage] { models.filter { $0.group == .cursor } }
    var otherModels: [ModelUsage] { models.filter { $0.group == .other } }

    var cursorSourceTokens: Double {
        models.filter { TokenSource.of($0.name) == .cursor }.reduce(0) { $0 + $1.totalTokens }
    }

    var grokBotSourceTokens: Double {
        models.filter { TokenSource.of($0.name) == .grokBot }.reduce(0) { $0 + $1.totalTokens }
    }

    var otherSourceTokens: Double {
        models.filter { TokenSource.of($0.name) == .other }.reduce(0) { $0 + $1.totalTokens }
    }

    func sourceUSD(_ source: TokenSource) -> Double? {
        let priced = models.filter { TokenSource.of($0.name) == source }.compactMap(\.estimatedUSD)
        return priced.isEmpty ? nil : priced.reduce(0, +)
    }

    var estimatedUSDTotal: Double {
        models.compactMap(\.estimatedUSD).reduce(0, +)
    }

    static func load(from defaults: UserDefaults) -> UsageSnapshot {
        SnapshotStore.load(defaults: defaults)
    }

    func save(to defaults: UserDefaults) {
        SnapshotStore.save(self, defaults: defaults)
    }

    init(
        monthlyLimit: Double,
        monthlyUsed: Double,
        todayUsed: Double,
        models: [ModelUsage],
        updatedAt: Date,
        includedPercent: Int,
        displayMessage: String,
        billingCycleStart: String,
        billingCycleEnd: String,
        planName: String,
        tokens: TokenTotals,
        todayTokens: TokenTotals,
        hasRealData: Bool,
        source: String,
        totalTokens: Double,
        includedTokens: Double,
        onDemandTokens: Double,
        cursorModelsPercent: Int?,
        otherModelsPercent: Int?,
        grokWeeklyPercent: Int?,
        grokResetAt: String,
        days: [DayUsage]
    ) {
        self.monthlyLimit = monthlyLimit
        self.monthlyUsed = monthlyUsed
        self.todayUsed = todayUsed
        self.models = models
        self.updatedAt = updatedAt
        self.includedPercent = includedPercent
        self.displayMessage = displayMessage
        self.billingCycleStart = billingCycleStart
        self.billingCycleEnd = billingCycleEnd
        self.planName = planName
        self.tokens = tokens
        self.todayTokens = todayTokens
        self.hasRealData = hasRealData
        self.source = source
        self.totalTokens = totalTokens
        self.includedTokens = includedTokens
        self.onDemandTokens = onDemandTokens
        self.cursorModelsPercent = cursorModelsPercent
        self.otherModelsPercent = otherModelsPercent
        self.grokWeeklyPercent = grokWeeklyPercent
        self.grokResetAt = grokResetAt
        self.days = days
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        monthlyLimit = try container.decodeIfPresent(Double.self, forKey: .monthlyLimit) ?? 0
        monthlyUsed = try container.decodeIfPresent(Double.self, forKey: .monthlyUsed) ?? 0
        todayUsed = try container.decodeIfPresent(Double.self, forKey: .todayUsed) ?? 0
        models = try container.decodeIfPresent([ModelUsage].self, forKey: .models) ?? []
        updatedAt = try container.decodeIfPresent(Date.self, forKey: .updatedAt) ?? .distantPast
        includedPercent = try container.decodeIfPresent(Int.self, forKey: .includedPercent) ?? 0
        displayMessage = try container.decodeIfPresent(String.self, forKey: .displayMessage) ?? ""
        billingCycleStart = try container.decodeIfPresent(String.self, forKey: .billingCycleStart) ?? ""
        billingCycleEnd = try container.decodeIfPresent(String.self, forKey: .billingCycleEnd) ?? ""
        planName = try container.decodeIfPresent(String.self, forKey: .planName) ?? ""
        tokens = try container.decodeIfPresent(TokenTotals.self, forKey: .tokens) ?? .zero
        todayTokens = try container.decodeIfPresent(TokenTotals.self, forKey: .todayTokens) ?? .zero
        hasRealData = try container.decodeIfPresent(Bool.self, forKey: .hasRealData) ?? false
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? ""
        totalTokens = try container.decodeIfPresent(Double.self, forKey: .totalTokens) ?? tokens.total
        includedTokens = try container.decodeIfPresent(Double.self, forKey: .includedTokens) ?? 0
        onDemandTokens = try container.decodeIfPresent(Double.self, forKey: .onDemandTokens) ?? 0
        cursorModelsPercent = try container.decodeIfPresent(Int.self, forKey: .cursorModelsPercent)
        otherModelsPercent = try container.decodeIfPresent(Int.self, forKey: .otherModelsPercent)
        grokWeeklyPercent = try container.decodeIfPresent(Int.self, forKey: .grokWeeklyPercent)
        grokResetAt = try container.decodeIfPresent(String.self, forKey: .grokResetAt) ?? ""
        days = try container.decodeIfPresent([DayUsage].self, forKey: .days) ?? []
    }
}

enum UsageJSON {
    static func number(_ value: Any?) -> Double? {
        if let number = value as? NSNumber { return number.doubleValue }
        if let text = value as? String {
            let cleaned = text.replacingOccurrences(of: "[^0-9.-]", with: "", options: .regularExpression)
            return Double(cleaned)
        }
        return nil
    }

    static func int(_ value: Any?) -> Int? {
        guard let number = number(value) else { return nil }
        return Int(number.rounded())
    }

    static func string(_ value: Any?) -> String? {
        if let text = value as? String, !text.isEmpty { return text }
        return nil
    }

    static func object(_ value: Any?) -> [String: Any]? {
        value as? [String: Any]
    }

    static func array(_ value: Any?) -> [Any]? {
        value as? [Any]
    }

    static func percent(from message: String?) -> Int? {
        guard let message else { return nil }
        guard let match = message.range(of: #"([0-9]+(?:\.[0-9]+)?)%"#, options: .regularExpression) else { return nil }
        return int(String(message[match]).replacingOccurrences(of: "%", with: ""))
    }

    static func toMs(_ value: Any?) -> Double? {
        if let number = number(value), number > 0 {
            return number < 1_000_000_000_000 ? number * 1000 : number
        }
        if let text = value as? String, let parsed = isoDate(text)?.timeIntervalSince1970 {
            return parsed * 1000
        }
        return nil
    }

    static func iso(_ ms: Double?) -> String {
        guard let ms else { return "" }
        return ISO8601DateFormatter().string(from: Date(timeIntervalSince1970: ms / 1000))
    }

    static func date(from text: String) -> Date? {
        if text.isEmpty { return nil }
        if let parsed = isoDate(text) { return parsed }
        if let ms = toMs(text) { return Date(timeIntervalSince1970: ms / 1000) }
        return nil
    }

    static func monthDay(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.dateFormat = "M月d日"
        return formatter.string(from: date)
    }

    static func utcDay(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
    }

    static func tokens(from value: Any?) -> TokenTotals {
        let source = object(object(value)?["tokenUsage"]) ?? object(value) ?? [:]
        var totals = TokenTotals(
            input: number(source["inputTokens"]) ?? 0,
            output: number(source["outputTokens"]) ?? 0,
            cacheRead: number(source["cacheReadTokens"]) ?? 0,
            cacheWrite: number(source["cacheWriteTokens"]) ?? 0,
            total: 0
        )
        totals.total = totals.input + totals.output + totals.cacheRead + totals.cacheWrite
        return totals
    }

    private static func isoDate(_ text: String) -> Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: text) { return date }
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: text)
    }
}

func formatTokens(_ value: Double) -> String {
    formatChineseCount(value)
}

func formatChineseCount(_ value: Double) -> String {
    if value >= 100_000_000 {
        let yi = value / 100_000_000
        if abs(yi - yi.rounded()) < 0.005 {
            return String(format: "%.0f亿", yi)
        }
        var text = String(format: "%.2f", yi)
        while text.contains(".") && (text.hasSuffix("0") || text.hasSuffix(".")) {
            text.removeLast()
        }
        return text + "亿"
    }
    if value >= 10_000 {
        let wan = value / 10_000
        if abs(wan - wan.rounded()) < 0.05 {
            return String(format: "%.0f万", wan)
        }
        return String(format: "%.1f万", wan)
    }
    return String(format: "%.0f", value)
}

struct PricingEntry: Codable, Equatable {
    var slug: String
    var requiresFast: Bool
    var input: Double
    var cacheWrite: Double
    var cacheRead: Double
    var output: Double
    var grok46Discount: Bool
}

struct PricingCatalog: Codable, Equatable {
    var syncedAt: Date
    var grok46DiscountEnd: Date?
    var entries: [PricingEntry]
}

enum PricingStore {
    private static let attemptKey = "pricingLastAttemptDay"
    private static let defaults = UserDefaults.standard

    static func localDay(_ date: Date = Date()) -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = Calendar.current.timeZone
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
    }

    static func needsSync(now: Date = Date()) -> Bool {
        defaults.string(forKey: attemptKey) != localDay(now)
    }

    static func markAttempted(now: Date = Date()) {
        defaults.set(localDay(now), forKey: attemptKey)
    }

    static func load() -> PricingCatalog? {
        for folder in SnapshotStore.storageFolders() {
            let url = folder.appendingPathComponent("pricing.json")
            if let data = try? Data(contentsOf: url),
               let catalog = try? JSONDecoder().decode(PricingCatalog.self, from: data),
               !catalog.entries.isEmpty {
                return catalog
            }
        }
        return nil
    }

    static func save(_ catalog: PricingCatalog) {
        guard let data = try? JSONEncoder().encode(catalog) else { return }
        for folder in SnapshotStore.storageFolders() {
            try? FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
            try? data.write(to: folder.appendingPathComponent("pricing.json"), options: .atomic)
        }
    }

    static func hasSameRates(_ lhs: PricingCatalog, _ rhs: PricingCatalog) -> Bool {
        lhs.entries == rhs.entries && lhs.grok46DiscountEnd == rhs.grok46DiscountEnd
    }

    @discardableResult
    static func saveIfChanged(_ catalog: PricingCatalog) -> Bool {
        if let current = load(), hasSameRates(current, catalog) { return false }
        save(catalog)
        return true
    }

    static var syncLabel: String {
        guard let catalog = load() else { return "标价用内置表" }
        return "标价已同步 \(UsageJSON.monthDay(catalog.syncedAt))"
    }
}

enum PricingParser {
    static func parse(_ markdown: String, now: Date = Date()) -> PricingCatalog? {
        var entries: [PricingEntry] = []
        var grokEnd: Date?
        let lines = markdown.components(separatedBy: .newlines)
        var index = 0
        while index < lines.count {
            if isPricingHeader(lines[index]) {
                index += 1
                if index < lines.count && lines[index].contains("---") { index += 1 }
                while index < lines.count {
                    let row = lines[index]
                    let trimmed = row.trimmingCharacters(in: .whitespaces)
                    if !trimmed.hasPrefix("|") { break }
                    if trimmed.contains("---") {
                        index += 1
                        continue
                    }
                    if let entry = parseRow(row) {
                        entries.append(entry)
                        if entry.grok46Discount, grokEnd == nil {
                            grokEnd = parseGrokEnd(from: row) ?? parseGrokEnd(from: markdown)
                        }
                    }
                    index += 1
                }
                continue
            }
            index += 1
        }
        guard !entries.isEmpty else { return nil }
        return PricingCatalog(syncedAt: now, grok46DiscountEnd: grokEnd, entries: entries)
    }

    private static func isPricingHeader(_ line: String) -> Bool {
        let lower = line.lowercased()
        return lower.contains("input")
            && lower.contains("cache write")
            && lower.contains("cache read")
            && lower.contains("output")
    }

    private static func parseRow(_ line: String) -> PricingEntry? {
        let cells = line.split(separator: "|", omittingEmptySubsequences: false)
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
        guard cells.count >= 6 else { return nil }
        let model = stripMarkdown(cells[0])
        guard let input = money(cells[2]), let output = money(cells[5]) else { return nil }
        let cacheWrite = money(cells[3]) ?? 0
        let cacheRead = money(cells[4]) ?? 0
        let notes = cells.count > 6 ? cells[6].lowercased() : ""
        let parsed = slug(from: model)
        guard !parsed.slug.isEmpty, parsed.slug != "model" else { return nil }
        let grok46 = parsed.slug.contains("grok-4.6") || notes.contains("50% launch discount")
        return PricingEntry(
            slug: parsed.slug,
            requiresFast: parsed.fast,
            input: input,
            cacheWrite: cacheWrite,
            cacheRead: cacheRead,
            output: output,
            grok46Discount: grok46
        )
    }

    static func slug(from title: String) -> (slug: String, fast: Bool) {
        var name = stripMarkdown(title).lowercased()
        let fast = name.contains("(fast") || name.contains("fast mode") || name.hasSuffix(" fast")
        for token in ["(fast mode)", "(fast)", "fast mode"] {
            name = name.replacingOccurrences(of: token, with: "")
        }
        if name.hasSuffix(" fast") { name = String(name.dropLast(5)) }
        if name.hasPrefix("claude ") { name = String(name.dropFirst(7)) }
        name = name.replacingOccurrences(of: #"\s+"#, with: " ", options: .regularExpression)
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return (name.replacingOccurrences(of: " ", with: "-"), fast)
    }

    static func parseGrokEnd(from text: String) -> Date? {
        let pattern = #"one week starting ([A-Za-z]+) (\d{1,2}), (\d{4})"#
        guard let match = text.range(of: pattern, options: [.regularExpression, .caseInsensitive]) else { return nil }
        let parts = String(text[match])
            .replacingOccurrences(of: "one week starting ", with: "", options: .caseInsensitive)
            .split(separator: " ")
        guard parts.count >= 3,
              let day = Int(parts[1].replacingOccurrences(of: ",", with: "")),
              let year = Int(parts[2]) else { return nil }
        let months = ["january": 1, "february": 2, "march": 3, "april": 4, "may": 5, "june": 6,
                      "july": 7, "august": 8, "september": 9, "october": 10, "november": 11, "december": 12]
        guard let month = months[parts[0].lowercased()] else { return nil }
        var components = DateComponents()
        components.calendar = Calendar(identifier: .gregorian)
        components.timeZone = TimeZone(secondsFromGMT: 0)
        components.year = year
        components.month = month
        components.day = day
        guard let start = components.date else { return nil }
        return start.addingTimeInterval(8 * 24 * 60 * 60)
    }

    private static func stripMarkdown(_ text: String) -> String {
        text.replacingOccurrences(of: #"\[([^\]]+)\]\([^)]+\)"#, with: "$1", options: .regularExpression)
            .replacingOccurrences(of: "**", with: "")
            .replacingOccurrences(of: "`", with: "")
    }

    private static func money(_ text: String) -> Double? {
        let trimmed = text.trimmingCharacters(in: .whitespaces)
        if trimmed == "-" || trimmed == "—" || trimmed.isEmpty { return 0 }
        let cleaned = trimmed.replacingOccurrences(of: "[^0-9.]", with: "", options: .regularExpression)
        return Double(cleaned)
    }
}

enum ModelPricing {
    struct Rate {
        var input: Double
        var cacheWrite: Double
        var cacheRead: Double
        var output: Double
        var grok46Discount: Bool = false
    }

    static func estimate(name: String, tokens: TokenTotals, now: Date = Date()) -> Double? {
        guard let rate = rate(for: name) else { return nil }
        var usd = tokens.input / 1_000_000 * rate.input
            + tokens.cacheWrite / 1_000_000 * rate.cacheWrite
            + tokens.cacheRead / 1_000_000 * rate.cacheRead
            + tokens.output / 1_000_000 * rate.output
        if rate.grok46Discount, now < grok46DiscountEnd { usd *= 0.5 }
        return usd
    }

    static func rate(for name: String) -> Rate? {
        let key = name.lowercased()
        if key.hasPrefix("sand-") || key == "default" { return nil }
        if let catalog = PricingStore.load(), let matched = match(name, in: catalog) {
            return matched
        }
        return bundledRate(for: key)
    }

    private static func match(_ name: String, in catalog: PricingCatalog) -> Rate? {
        let usage = normalizeUsage(name)
        let wantFast = usage.contains("fast")
        let hits = catalog.entries.filter { entry in
            usage.contains(entry.slug) && entry.requiresFast == wantFast
        }
        guard let best = hits.max(by: { $0.slug.count < $1.slug.count }) else { return nil }
        return Rate(
            input: best.input,
            cacheWrite: best.cacheWrite,
            cacheRead: best.cacheRead,
            output: best.output,
            grok46Discount: best.grok46Discount
        )
    }

    private static func normalizeUsage(_ name: String) -> String {
        var key = name.lowercased()
        if key.hasPrefix("cursor-") { key = String(key.dropFirst(7)) }
        if key.hasPrefix("claude-") { key = String(key.dropFirst(7)) }
        return key.replacingOccurrences(of: #"(\d)-(\d)"#, with: "$1.$2", options: .regularExpression)
    }

    private static func bundledRate(for key: String) -> Rate? {
        if key.contains("composer-2.5") && key.contains("fast") {
            return Rate(input: 3, cacheWrite: 0, cacheRead: 0.5, output: 15)
        }
        if key.contains("composer-2.5") {
            return Rate(input: 0.5, cacheWrite: 0, cacheRead: 0.2, output: 2.5)
        }
        if key.contains("grok-4.6") && key.contains("fast") {
            return Rate(input: 4, cacheWrite: 0, cacheRead: 1, output: 12, grok46Discount: true)
        }
        if key.contains("grok-4.6") {
            return Rate(input: 2, cacheWrite: 0, cacheRead: 0.5, output: 6, grok46Discount: true)
        }
        if key.contains("grok-4.5") && key.contains("fast") {
            return Rate(input: 4, cacheWrite: 0, cacheRead: 1, output: 12)
        }
        if key.contains("grok-4.5") {
            return Rate(input: 2, cacheWrite: 0, cacheRead: 0.5, output: 6)
        }
        if key.contains("opus-4-8") || key.contains("opus-4.8") {
            return Rate(input: 5, cacheWrite: 6.25, cacheRead: 0.5, output: 25)
        }
        if key.contains("opus-5") {
            return Rate(input: 5, cacheWrite: 6.25, cacheRead: 0.5, output: 25)
        }
        if key.contains("4.5-sonnet") || key.contains("sonnet-4.5")
            || key.contains("4-5-sonnet") || key.contains("sonnet-4-5") {
            return Rate(input: 3, cacheWrite: 3.75, cacheRead: 0.3, output: 15)
        }
        if key.contains("gemini-3.1-pro") {
            return Rate(input: 2, cacheWrite: 0, cacheRead: 0.2, output: 12)
        }
        return nil
    }

    static var grok46DiscountEnd: Date {
        if let cached = PricingStore.load()?.grok46DiscountEnd { return cached }
        var parts = DateComponents()
        parts.calendar = Calendar(identifier: .gregorian)
        parts.timeZone = TimeZone(secondsFromGMT: 0)
        parts.year = 2026
        parts.month = 8
        parts.day = 20
        return parts.date ?? .distantPast
    }
}

func formatUSD(_ value: Double) -> String {
    if value >= 100 { return String(format: "$%.0f", value) }
    if value >= 10 { return String(format: "$%.1f", value) }
    return String(format: "$%.2f", value)
}

func formatEstimate(_ value: Double?) -> String {
    guard let value else { return "无标价" }
    return "预估 " + formatUSD(value)
}

func formatWan(_ value: Double) -> String {
    formatChineseCount(value)
}

func displayModelName(_ raw: String) -> String {
    var name = raw
    let lower = name.lowercased()
    if lower.hasPrefix("cursor-") { name = String(name.dropFirst(7)) }
    else if lower.hasPrefix("claude-") { name = String(name.dropFirst(7)) }
    name = name.replacingOccurrences(of: "thinking-", with: "", options: .caseInsensitive)
    name = name.replacingOccurrences(of: "-thinking", with: "", options: .caseInsensitive)
    name = name.replacingOccurrences(of: #"(\d)-(\d)"#, with: "$1.$2", options: .regularExpression)
    let key = name.lowercased()
    if key.hasPrefix("grok-") { return "Grok " + spaced(String(name.dropFirst(5))) }
    if key.hasPrefix("composer-") { return "Composer " + spaced(String(name.dropFirst(9))) }
    if key.hasPrefix("opus-") { return "Opus " + spaced(String(name.dropFirst(5))) }
    if key.hasPrefix("gemini-") { return "Gemini " + spaced(String(name.dropFirst(7))) }
    if key.hasPrefix("sand-") { return "Sand " + spaced(String(name.dropFirst(5))) }
    return spaced(name)
}

private func spaced(_ text: String) -> String {
    text.replacingOccurrences(of: "-", with: " ")
}
