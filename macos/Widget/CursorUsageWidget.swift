import SwiftUI
import WidgetKit

struct UsageEntry: TimelineEntry {
    let date: Date
    let snapshot: UsageSnapshot
}

struct Provider: TimelineProvider {
    func placeholder(in context: Context) -> UsageEntry {
        .init(date: .now, snapshot: .sample)
    }

    func getSnapshot(in context: Context, completion: @escaping (UsageEntry) -> Void) {
        completion(.init(date: .now, snapshot: SnapshotStore.load()))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<UsageEntry>) -> Void) {
        let entry = UsageEntry(date: .now, snapshot: SnapshotStore.load())
        completion(Timeline(entries: [entry], policy: .after(.now.addingTimeInterval(300))))
    }
}

struct CursorUsageWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: UsageEntry

    var body: some View {
        panel
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .widgetAccentable()
            .containerBackground(.background, for: .widget)
    }

    @ViewBuilder
    private var panel: some View {
        switch family {
        case .systemSmall:
            smallLayout
        case .systemMedium:
            mediumLayout
        default:
            largeLayout
        }
    }

    private var snap: UsageSnapshot { entry.snapshot }

    private var smallLayout: some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack(spacing: 6) {
                Text("用量")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(1)
                Spacer(minLength: 4)
                WidgetPlanBadge(title: planTitle, size: 11)
            }
            Text(percentText)
                .font(.system(size: 28, weight: .bold, design: .rounded))
                .foregroundStyle(.primary)
                .monospacedDigit()
                .lineLimit(1)
                .minimumScaleFactor(0.7)
            Text("剩余 · Cursor")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Text("总消耗")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                Text(monthTokens)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(1)
            }
            Text(cycleShort)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var mediumLayout: some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack(spacing: 6) {
                Text("用量")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(1)
                WidgetPlanBadge(title: planTitle, size: 11)
                Spacer(minLength: 6)
                Text(cycleShort)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
            }
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Text(percentText)
                    .font(.system(size: 28, weight: .bold, design: .rounded))
                    .foregroundStyle(.primary)
                    .monospacedDigit()
                Text("剩余 · Cursor")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(1)
            }
            metricGrid(valueSize: 17, rowSpacing: 6)
            Text(quotaLine)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var largeLayout: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Text("Cursor 用量")
                    .font(.headline)
                    .foregroundStyle(.primary)
                    .lineLimit(1)
                    .layoutPriority(1)
                Spacer(minLength: 8)
                WidgetPlanBadge(title: planTitle, size: 13)
            }
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(percentText)
                    .font(.system(size: 36, weight: .bold, design: .rounded))
                    .foregroundStyle(.primary)
                    .monospacedDigit()
                VStack(alignment: .leading, spacing: 2) {
                    Text("剩余 · Cursor 模型")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.primary)
                    Text(cycleShort)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer(minLength: 0)
            metricGrid(valueSize: 22, rowSpacing: 10)
            Spacer(minLength: 0)
            Text(quotaLine)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(1)
            compactBar(title: "Cursor", percent: snap.cursorModelsPercent)
            compactBar(title: "其他", percent: snap.otherModelsPercent)
            compactBar(title: "Grok", percent: snap.grokWeeklyPercent)
            Text(footer)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private func metricGrid(valueSize: CGFloat, rowSpacing: CGFloat) -> some View {
        VStack(spacing: rowSpacing) {
            HStack(alignment: .top, spacing: 12) {
                metric(title: "今日 token", value: formatTokens(snap.todayTokens.total), valueSize: valueSize)
                metric(title: "本月 token", value: monthTokens, valueSize: valueSize)
            }
            HStack(alignment: .top, spacing: 12) {
                metric(title: "Cursor 基础", value: formatEstimate(snap.sourceUSD(.cursor)), valueSize: valueSize)
                metric(title: "高级模型", value: formatEstimate(snap.sourceUSD(.other)), valueSize: valueSize)
            }
        }
    }

    private func metric(title: String, value: String, valueSize: CGFloat) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.system(size: valueSize, weight: .semibold, design: .rounded))
                .foregroundStyle(.primary)
                .monospacedDigit()
                .lineLimit(1)
                .minimumScaleFactor(0.75)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func compactBar(title: String, percent: Int?) -> some View {
        let value = min(max(percent ?? 0, 0), 100)
        return HStack(spacing: 8) {
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: 48, alignment: .leading)
            HStack(spacing: 0) {
                Capsule()
                    .fill(.primary)
                    .frame(maxWidth: .infinity)
                    .layoutPriority(Double(value))
                Color.clear
                    .frame(maxWidth: .infinity)
                    .layoutPriority(Double(100 - value))
            }
            .frame(height: 6)
            .background(Capsule().fill(.quaternary))
            .clipShape(Capsule())
            Text(percent.map { "\($0)%" } ?? "—")
                .font(.caption.monospacedDigit())
                .foregroundStyle(.primary)
                .frame(width: 32, alignment: .trailing)
        }
    }

    private var planTitle: String {
        snap.hasRealData ? snap.planLabel : "未连接"
    }

    private var percentText: String {
        snap.hasRealData ? "\(snap.remainingPercent)%" : "—"
    }

    private var monthTokens: String {
        formatTokens(snap.totalTokens > 0 ? snap.totalTokens : snap.tokens.total)
    }

    private var quotaLine: String {
        let cursor = snap.cursorModelsPercent.map { "Cursor \($0)%" } ?? "Cursor —"
        let other = snap.otherModelsPercent.map { "其他 \($0)%" } ?? "其他 —"
        let grok = snap.grokWeeklyPercent.map { "Grok \($0)%" } ?? "Grok —"
        return "\(cursor) · \(other) · \(grok)"
    }

    private var cycleShort: String {
        if !snap.hasRealData { return "未连接" }
        if let days = snap.daysRemaining {
            let cycle = snap.cycleEndLabel.isEmpty ? "" : " · \(snap.cycleEndLabel)"
            return "还剩 \(days) 天\(cycle)"
        }
        return footer
    }

    private var footer: String {
        if !snap.hasRealData { return "打开主应用并粘贴 Cookie" }
        if let days = snap.daysRemaining {
            let cycle = snap.cycleEndLabel.isEmpty ? "" : " · \(snap.cycleEndLabel)"
            return "账期还剩 \(days) 天\(cycle)"
        }
        return "已更新 \(snap.updatedAt.formatted(date: .omitted, time: .shortened))"
    }
}

struct WidgetPlanBadge: View {
    let title: String
    var size: CGFloat = 12

    var body: some View {
        Text(title)
            .font(.system(size: size, weight: .bold))
            .foregroundStyle(.primary)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(.fill, in: Capsule())
            .fixedSize()
            .layoutPriority(1)
    }
}

@main
struct CursorUsageWidget: Widget {
    let kind = "CursorUsageWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: Provider()) { CursorUsageWidgetView(entry: $0) }
            .configurationDisplayName("Cursor 用量")
            .description("显示本机保存的 Cursor token 余量和账期。")
            .supportedFamilies([.systemSmall, .systemMedium, .systemLarge])
    }
}
