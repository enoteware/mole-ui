// PrivilegedHelper - Runs shell commands with admin privileges
// Uses NSAppleScript to trigger macOS native auth dialog
// Usage: privileged-helper <command> [args...]

import Foundation

// Main entry point
let args = CommandLine.arguments

guard args.count >= 2 else {
    fputs("Usage: privileged-helper <command> [args...]\n", stderr)
    fputs("Example: privileged-helper /bin/rm -rf /path/to/file\n", stderr)
    exit(1)
}

// Build the command string
let command = args[1]
let cmdArgs = Array(args.dropFirst(2))

// Escape for AppleScript
func escapeForAppleScript(_ str: String) -> String {
    return str
        .replacingOccurrences(of: "\\", with: "\\\\")
        .replacingOccurrences(of: "\"", with: "\\\"")
}

// Shell-escape arguments so spaces/special characters survive the shell parse.
func shellEscape(_ str: String) -> String {
    let escaped = str.replacingOccurrences(of: "'", with: "'\\''")
    return "'" + escaped + "'"
}

// Build the shell command with safe shell quoting, then escape for AppleScript.
var shellCmd = shellEscape(command)
for arg in cmdArgs {
    shellCmd += " " + shellEscape(arg)
}
shellCmd = escapeForAppleScript(shellCmd)

// Create AppleScript that runs command with admin privileges
let script = """
do shell script "\(shellCmd)" with administrator privileges
"""

var error: NSDictionary?
if let scriptObject = NSAppleScript(source: script) {
    let result = scriptObject.executeAndReturnError(&error)

    if let error = error {
        let errorNum = error[NSAppleScript.errorNumber] as? Int ?? -1
        let errorMsg = error[NSAppleScript.errorMessage] as? String ?? "Unknown error"

        // -128 is user cancelled
        if errorNum == -128 {
            fputs("Authorization cancelled by user\n", stderr)
        } else {
            fputs("Error (\(errorNum)): \(errorMsg)\n", stderr)
        }
        exit(1)
    }

    // Success
    exit(0)
} else {
    fputs("Error: Failed to create AppleScript\n", stderr)
    exit(1)
}
