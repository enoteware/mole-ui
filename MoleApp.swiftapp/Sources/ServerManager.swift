import Foundation
import Combine

class ServerManager: ObservableObject {
    @Published var isServerRunning = false
    @Published var isStarting = false
    @Published var error: String?
    
    private var serverProcess: Process?
    private var healthCheckTimer: Timer?
    
    func startServer() {
        guard serverProcess == nil else { return }

        isStarting = true
        error = nil

        // Get the path to the bundled binary
        let appPath = Bundle.main.bundlePath
        print("App bundle path: \(appPath)")

        // Try multiple possible locations for the binary
        let possiblePaths = [
            // Standard app bundle location
            "\(appPath)/Contents/MacOS/web-go",
            // Development/preview location - check in project directory
            "\(FileManager.default.currentDirectoryPath)/bin/web-go",
            // Relative to app bundle for .swiftapp packages
            URL(fileURLWithPath: appPath).deletingLastPathComponent().appendingPathComponent("bin/web-go").path,
            // Check parent directories for bin/web-go
            URL(fileURLWithPath: appPath).deletingLastPathComponent().deletingLastPathComponent().appendingPathComponent("bin/web-go").path
        ]

        var binaryPath: String?
        let fileManager = FileManager.default

        for path in possiblePaths {
            print("Checking for binary at: \(path)")
            if fileManager.fileExists(atPath: path) {
                binaryPath = path
                print("Found binary at: \(path)")
                break
            }
        }

        guard let finalPath = binaryPath else {
            error = "Server binary not found. Checked paths:\n\(possiblePaths.joined(separator: "\n"))"
            isStarting = false
            return
        }
        
        // Set up process
        let process = Process()
        process.executableURL = URL(fileURLWithPath: finalPath)
        
        // Set environment variables
        var environment = ProcessInfo.processInfo.environment
        environment["MOLE_PORT"] = "8081"
        environment["MOLE_HOST"] = "127.0.0.1"
        environment["MOLE_NO_OPEN"] = "1"
        process.environment = environment
        
        // Redirect output to log file
        let logPath = getLogPath()
        if let logURL = URL(string: "file://\(logPath)") {
            let logFile = try? FileHandle(forWritingTo: logURL)
            process.standardOutput = logFile
            process.standardError = logFile
        }
        
        do {
            try process.run()
            serverProcess = process
            
            // Start health checks
            startHealthChecks()
        } catch {
            self.error = "Failed to start server: \(error.localizedDescription)"
            isStarting = false
        }
    }
    
    func stopServer() {
        healthCheckTimer?.invalidate()
        healthCheckTimer = nil
        
        serverProcess?.terminate()
        serverProcess = nil
        isServerRunning = false
    }
    
    private func startHealthChecks() {
        var attempts = 0
        let maxAttempts = 20
        
        healthCheckTimer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { [weak self] timer in
            guard let self = self else {
                timer.invalidate()
                return
            }
            
            attempts += 1
            
            self.checkServerHealth { isHealthy in
                if isHealthy {
                    DispatchQueue.main.async {
                        self.isServerRunning = true
                        self.isStarting = false
                        timer.invalidate()
                    }
                } else if attempts >= maxAttempts {
                    DispatchQueue.main.async {
                        self.error = "Server failed to start after \(maxAttempts) attempts"
                        self.isStarting = false
                        timer.invalidate()
                    }
                }
            }
        }
    }
    
    private func checkServerHealth(completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "http://127.0.0.1:8081/health") else {
            completion(false)
            return
        }
        
        let task = URLSession.shared.dataTask(with: url) { data, response, error in
            if let httpResponse = response as? HTTPURLResponse,
               httpResponse.statusCode == 200 {
                completion(true)
            } else {
                completion(false)
            }
        }
        task.resume()
    }
    
    private func getLogPath() -> String {
        let supportDir = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
        let moleDir = supportDir.appendingPathComponent("Mole", isDirectory: true)
        
        try? FileManager.default.createDirectory(at: moleDir, withIntermediateDirectories: true)
        
        return moleDir.appendingPathComponent("server.log").path
    }
    
    deinit {
        stopServer()
    }
}
