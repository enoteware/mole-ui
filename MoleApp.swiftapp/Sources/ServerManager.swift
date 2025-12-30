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
        guard let appPath = Bundle.main.bundlePath as NSString? else {
            error = "Could not find app bundle"
            isStarting = false
            return
        }
        
        let contentsPath = appPath.appendingPathComponent("Contents")
        let macOSPath = (contentsPath as NSString).appendingPathComponent("MacOS")
        let binaryPath = (macOSPath as NSString).appendingPathComponent("web-go")
        
        // Check if binary exists
        let fileManager = FileManager.default
        guard fileManager.fileExists(atPath: binaryPath) else {
            error = "Server binary not found at: \(binaryPath)"
            isStarting = false
            return
        }
        
        // Set up process
        let process = Process()
        process.executableURL = URL(fileURLWithPath: binaryPath)
        
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
