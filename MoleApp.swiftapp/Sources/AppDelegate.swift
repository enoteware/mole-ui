import AppKit
import SwiftUI

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_ notification: Notification) {
        maybeOfferMoveToApplications()

        // Configure window
        if let window = NSApplication.shared.windows.first {
            window.titlebarAppearsTransparent = true
            window.titleVisibility = .hidden
            window.setContentSize(NSSize(width: 1200, height: 800))
            window.minSize = NSSize(width: 800, height: 600)
            window.center()
        }
    }
    
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        // Server cleanup is handled by ServerManager deinit
    }

    private func maybeOfferMoveToApplications() {
        let appURL = Bundle.main.bundleURL
        let appPath = appURL.path

        guard shouldOfferMoveToApplications(appPath: appPath) else { return }

        let alert = NSAlert()
        alert.alertStyle = .informational
        alert.messageText = "Move Mole to Applications?"
        alert.informativeText = "For best performance and updates, Mole should live in the Applications folder."
        alert.addButton(withTitle: "Move to Applications")
        alert.addButton(withTitle: "Continue")

        let response = alert.runModal()
        if response == .alertFirstButtonReturn {
            moveToApplications(appURL: appURL)
        }
    }

    private func shouldOfferMoveToApplications(appPath: String) -> Bool {
        let env = ProcessInfo.processInfo.environment
        if env["MOLE_SKIP_INSTALL_CHECK"] == "1" {
            return false
        }

        if appPath.hasPrefix("/Applications/") {
            return false
        }

        let home = FileManager.default.homeDirectoryForCurrentUser.path
        if appPath.hasPrefix("\(home)/Applications/") {
            return false
        }

        if appPath.contains("AppTranslocation") {
            return true
        }

        let commonPrefixes = [
            "/Volumes/",
            "\(home)/Downloads/",
            "\(home)/Desktop/",
            "\(home)/Documents/",
        ]

        return commonPrefixes.contains(where: { appPath.hasPrefix($0) })
    }

    private func moveToApplications(appURL: URL) {
        let applicationsURL = URL(fileURLWithPath: "/Applications", isDirectory: true)
        let targetURL = applicationsURL.appendingPathComponent(appURL.lastPathComponent)

        if appURL.standardizedFileURL == targetURL.standardizedFileURL {
            return
        }

        guard FileManager.default.isWritableFile(atPath: applicationsURL.path) else {
            showMoveFailed(message: "Please drag Mole.app into Applications manually and open it again.")
            return
        }

        do {
            if FileManager.default.fileExists(atPath: targetURL.path) {
                try FileManager.default.removeItem(at: targetURL)
            }

            try FileManager.default.copyItem(at: appURL, to: targetURL)

            let config = NSWorkspace.OpenConfiguration()
            NSWorkspace.shared.openApplication(at: targetURL, configuration: config) { _, _ in
                NSApp.terminate(nil)
            }
        } catch {
            showMoveFailed(message: "Please drag Mole.app into Applications manually and open it again.")
        }
    }

    private func showMoveFailed(message: String) {
        let alert = NSAlert()
        alert.alertStyle = .warning
        alert.messageText = "Couldn't move Mole"
        alert.informativeText = message
        alert.addButton(withTitle: "Open Applications Folder")
        alert.addButton(withTitle: "OK")

        let response = alert.runModal()
        if response == .alertFirstButtonReturn {
            NSWorkspace.shared.open(URL(fileURLWithPath: "/Applications", isDirectory: true))
        }
    }
}
