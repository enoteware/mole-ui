import WidgetKit
import SwiftUI
import Foundation
import Darwin

// MARK: - Memory Data Types (shared with main app)

struct WidgetMemoryStats {
    let total: UInt64
    let used: UInt64
    let free: UInt64
    let wired: UInt64
    let active: UInt64
    let inactive: UInt64
    let compressed: UInt64
    let pressure: WidgetMemoryPressure
    
    var usedPercentage: Double {
        guard total > 0 else { return 0 }
        return Double(used) / Double(total) * 100
    }
}

enum WidgetMemoryPressure: String {
    case normal = "Normal"
    case warn = "Pressure"
    case critical = "Critical"
    
    var color: Color {
        switch self {
        case .normal: return .green
        case .warn: return .yellow
        case .critical: return .red
        }
    }
    
    var systemImage: String {
        switch self {
        case .normal: return "checkmark.circle.fill"
        case .warn: return "exclamationmark.triangle.fill"
        case .critical: return "xmark.circle.fill"
        }
    }
}

// MARK: - Memory Provider for Widget

struct WidgetMemoryProvider {
    static func getMemoryStats() -> WidgetMemoryStats {
        var stats = vm_statistics64()
        var count = mach_msg_type_number_t(MemoryLayout<vm_statistics64>.size / MemoryLayout<integer_t>.size)
        
        let result = withUnsafeMutablePointer(to: &stats) { statsPtr in
            statsPtr.withMemoryRebound(to: integer_t.self, capacity: Int(count)) { intPtr in
                host_statistics64(mach_host_self(), HOST_VM_INFO64, intPtr, &count)
            }
        }
        
        guard result == KERN_SUCCESS else {
            return WidgetMemoryStats(
                total: 0, used: 0, free: 0, wired: 0,
                active: 0, inactive: 0, compressed: 0,
                pressure: .normal
            )
        }
        
        let pageSize = UInt64(vm_kernel_page_size)
        let total = ProcessInfo.processInfo.physicalMemory
        
        let free = UInt64(stats.free_count) * pageSize
        let wired = UInt64(stats.wire_count) * pageSize
        let active = UInt64(stats.active_count) * pageSize
        let inactive = UInt64(stats.inactive_count) * pageSize
        let compressed = UInt64(stats.compressor_page_count) * pageSize
        
        let used = wired + active + compressed
        
        // Determine pressure
        let availableRatio = Double(free + inactive) / Double(total)
        let pressure: WidgetMemoryPressure
        if availableRatio < 0.05 {
            pressure = .critical
        } else if availableRatio < 0.15 {
            pressure = .warn
        } else {
            pressure = .normal
        }
        
        return WidgetMemoryStats(
            total: total,
            used: used,
            free: free,
            wired: wired,
            active: active,
            inactive: inactive,
            compressed: compressed,
            pressure: pressure
        )
    }
    
    static func formatBytes(_ bytes: UInt64) -> String {
        let gb = Double(bytes) / 1_073_741_824
        if gb >= 1.0 {
            return String(format: "%.1f GB", gb)
        }
        let mb = Double(bytes) / 1_048_576
        return String(format: "%.0f MB", mb)
    }
}

// MARK: - Timeline Entry

struct MemoryEntry: TimelineEntry {
    let date: Date
    let stats: WidgetMemoryStats
}

// MARK: - Timeline Provider

struct MemoryTimelineProvider: TimelineProvider {
    func placeholder(in context: Context) -> MemoryEntry {
        MemoryEntry(date: Date(), stats: WidgetMemoryProvider.getMemoryStats())
    }
    
    func getSnapshot(in context: Context, completion: @escaping (MemoryEntry) -> Void) {
        let entry = MemoryEntry(date: Date(), stats: WidgetMemoryProvider.getMemoryStats())
        completion(entry)
    }
    
    func getTimeline(in context: Context, completion: @escaping (Timeline<MemoryEntry>) -> Void) {
        let currentDate = Date()
        let stats = WidgetMemoryProvider.getMemoryStats()
        let entry = MemoryEntry(date: currentDate, stats: stats)
        
        // Refresh every 5 minutes
        let nextUpdate = Calendar.current.date(byAdding: .minute, value: 5, to: currentDate)!
        let timeline = Timeline(entries: [entry], policy: .after(nextUpdate))
        completion(timeline)
    }
}

// MARK: - Widget View

struct MemoryWidgetView: View {
    var entry: MemoryEntry
    @Environment(\.widgetFamily) var family
    
    var body: some View {
        switch family {
        case .systemSmall:
            SmallMemoryView(entry: entry)
        case .systemMedium:
            MediumMemoryView(entry: entry)
        default:
            SmallMemoryView(entry: entry)
        }
    }
}

struct SmallMemoryView: View {
    var entry: MemoryEntry
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Header
            HStack {
                Image(systemName: "memorychip")
                    .foregroundColor(.secondary)
                Text("Memory")
                    .font(.caption)
                    .fontWeight(.semibold)
                    .foregroundColor(.secondary)
                Spacer()
            }
            
            Spacer()
            
            // Pressure indicator
            HStack(spacing: 6) {
                Image(systemName: entry.stats.pressure.systemImage)
                    .foregroundColor(entry.stats.pressure.color)
                    .font(.title2)
                Text(entry.stats.pressure.rawValue)
                    .font(.headline)
                    .fontWeight(.medium)
            }
            
            // Usage
            VStack(alignment: .leading, spacing: 4) {
                Text("\(WidgetMemoryProvider.formatBytes(entry.stats.used)) used")
                    .font(.caption)
                    .foregroundColor(.primary)
                
                // Progress bar
                GeometryReader { geometry in
                    ZStack(alignment: .leading) {
                        RoundedRectangle(cornerRadius: 3)
                            .fill(Color.gray.opacity(0.3))
                            .frame(height: 6)
                        
                        RoundedRectangle(cornerRadius: 3)
                            .fill(entry.stats.pressure.color)
                            .frame(width: geometry.size.width * min(entry.stats.usedPercentage / 100, 1.0), height: 6)
                    }
                }
                .frame(height: 6)
                
                Text("of \(WidgetMemoryProvider.formatBytes(entry.stats.total))")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
        }
        .padding()
    }
}

struct MediumMemoryView: View {
    var entry: MemoryEntry
    
    var body: some View {
        HStack(spacing: 16) {
            // Left side - gauge
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Image(systemName: "memorychip")
                        .foregroundColor(.secondary)
                    Text("Memory")
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.secondary)
                }
                
                Spacer()
                
                // Circular gauge
                ZStack {
                    Circle()
                        .stroke(Color.gray.opacity(0.3), lineWidth: 8)
                    
                    Circle()
                        .trim(from: 0, to: min(entry.stats.usedPercentage / 100, 1.0))
                        .stroke(entry.stats.pressure.color, style: StrokeStyle(lineWidth: 8, lineCap: .round))
                        .rotationEffect(.degrees(-90))
                    
                    VStack(spacing: 2) {
                        Text("\(Int(entry.stats.usedPercentage))%")
                            .font(.title3)
                            .fontWeight(.bold)
                        Image(systemName: entry.stats.pressure.systemImage)
                            .foregroundColor(entry.stats.pressure.color)
                            .font(.caption)
                    }
                }
                .frame(width: 70, height: 70)
                
                Spacer()
            }
            
            // Right side - breakdown
            VStack(alignment: .leading, spacing: 6) {
                MemoryRow(label: "Used", value: entry.stats.used, color: entry.stats.pressure.color)
                MemoryRow(label: "Wired", value: entry.stats.wired, color: .orange)
                MemoryRow(label: "Compressed", value: entry.stats.compressed, color: .purple)
                MemoryRow(label: "Free", value: entry.stats.free + entry.stats.inactive, color: .green)
                
                Divider()
                
                HStack {
                    Text("Total")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Spacer()
                    Text(WidgetMemoryProvider.formatBytes(entry.stats.total))
                        .font(.caption)
                        .fontWeight(.medium)
                }
            }
            .frame(maxWidth: .infinity)
        }
        .padding()
    }
}

struct MemoryRow: View {
    let label: String
    let value: UInt64
    let color: Color
    
    var body: some View {
        HStack {
            Circle()
                .fill(color)
                .frame(width: 8, height: 8)
            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
            Spacer()
            Text(WidgetMemoryProvider.formatBytes(value))
                .font(.caption)
                .fontWeight(.medium)
        }
    }
}

// MARK: - Widget Configuration

struct MemoryWidget: Widget {
    let kind: String = "MemoryWidget"
    
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: MemoryTimelineProvider()) { entry in
            MemoryWidgetView(entry: entry)
                .containerBackground(.fill.tertiary, for: .widget)
        }
        .configurationDisplayName("Memory Monitor")
        .description("Shows system memory usage and pressure.")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}

// MARK: - Preview

#Preview(as: .systemSmall) {
    MemoryWidget()
} timeline: {
    MemoryEntry(date: .now, stats: WidgetMemoryProvider.getMemoryStats())
}

#Preview(as: .systemMedium) {
    MemoryWidget()
} timeline: {
    MemoryEntry(date: .now, stats: WidgetMemoryProvider.getMemoryStats())
}
