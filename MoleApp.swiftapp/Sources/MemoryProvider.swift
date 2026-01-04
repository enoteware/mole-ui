import Foundation
import Darwin

/// System memory statistics
struct MemoryStats {
    let total: UInt64           // Total physical memory
    let used: UInt64            // Used memory (total - free - inactive)
    let free: UInt64            // Free memory
    let wired: UInt64           // Wired (non-pageable) memory
    let active: UInt64          // Recently accessed memory
    let inactive: UInt64        // Not recently accessed
    let compressed: UInt64      // Compressed memory
    let pressure: MemoryPressure
    
    var usedPercentage: Double {
        guard total > 0 else { return 0 }
        return Double(used) / Double(total) * 100
    }
    
    var freePercentage: Double {
        guard total > 0 else { return 0 }
        return Double(free + inactive) / Double(total) * 100
    }
}

/// Memory pressure levels matching macOS Activity Monitor
enum MemoryPressure: String, Codable {
    case normal = "Normal"
    case warn = "Pressure"
    case critical = "Critical"
    
    var color: String {
        switch self {
        case .normal: return "green"
        case .warn: return "yellow"
        case .critical: return "red"
        }
    }
}

/// Provider for system memory statistics using Mach APIs
class MemoryProvider {
    static let shared = MemoryProvider()
    
    private init() {}
    
    /// Get current system memory statistics
    func getMemoryStats() -> MemoryStats {
        var stats = vm_statistics64()
        var count = mach_msg_type_number_t(MemoryLayout<vm_statistics64>.size / MemoryLayout<integer_t>.size)
        
        let result = withUnsafeMutablePointer(to: &stats) { statsPtr in
            statsPtr.withMemoryRebound(to: integer_t.self, capacity: Int(count)) { intPtr in
                host_statistics64(mach_host_self(), HOST_VM_INFO64, intPtr, &count)
            }
        }
        
        guard result == KERN_SUCCESS else {
            return MemoryStats(
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
        
        // Used = wired + active + compressed (similar to Activity Monitor)
        let used = wired + active + compressed
        
        // Determine memory pressure
        let pressure = getMemoryPressure(stats: stats, total: total)
        
        return MemoryStats(
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
    
    /// Determine memory pressure level
    private func getMemoryPressure(stats: vm_statistics64, total: UInt64) -> MemoryPressure {
        let pageSize = UInt64(vm_kernel_page_size)
        
        // Calculate memory pressure based on available pages vs total
        let freePages = UInt64(stats.free_count)
        let inactivePages = UInt64(stats.inactive_count)
        let availablePages = freePages + inactivePages
        let totalPages = total / pageSize
        
        guard totalPages > 0 else { return .normal }
        
        let availableRatio = Double(availablePages) / Double(totalPages)
        
        // Thresholds based on typical macOS behavior
        if availableRatio < 0.05 {
            return .critical
        } else if availableRatio < 0.15 {
            return .warn
        } else {
            return .normal
        }
    }
    
    /// Format bytes to human-readable string (e.g., "8.5 GB")
    static func formatBytes(_ bytes: UInt64) -> String {
        let gb = Double(bytes) / 1_073_741_824
        if gb >= 1.0 {
            return String(format: "%.1f GB", gb)
        }
        let mb = Double(bytes) / 1_048_576
        return String(format: "%.0f MB", mb)
    }
}
