import AppKit
import CoreGraphics
import Foundation

struct WatchWindow {
    let id: UInt32
    let owner: String
    let title: String

    var label: String {
        if title.isEmpty {
            return "\(owner) (#\(id))"
        }
        return "\(owner): \(title)"
    }
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let menu = NSMenu()
    private let windowsMenu = NSMenu()
    private var selectedWindow: WatchWindow?
    private var watcher: Process?
    private var selectedItem = NSMenuItem(title: "Selected: none", action: nil, keyEquivalent: "")
    private var startItem = NSMenuItem(title: "Start Watching", action: #selector(startWatching), keyEquivalent: "s")
    private var stopItem = NSMenuItem(title: "Stop Watching", action: #selector(stopWatching), keyEquivalent: "x")
    private var logItem = NSMenuItem(title: "Open Log", action: #selector(openLog), keyEquivalent: "l")
    private var statusItemRow = NSMenuItem(title: "Status: idle", action: nil, keyEquivalent: "")
    private let logURL = URL(fileURLWithPath: NSTemporaryDirectory()).appendingPathComponent("opswatch-menubar.log")
    private var logHandle: FileHandle?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
        setStatus(.idle)
        configureMenu()
        refreshWindows()
    }

    private func configureMenu() {
        statusItemRow.isEnabled = false
        menu.addItem(statusItemRow)

        selectedItem.isEnabled = false
        menu.addItem(selectedItem)

        let chooseItem = NSMenuItem(title: "Windows", action: nil, keyEquivalent: "")
        chooseItem.submenu = windowsMenu
        menu.addItem(chooseItem)

        menu.addItem(NSMenuItem(title: "Refresh Windows", action: #selector(refreshWindows), keyEquivalent: "r"))
        menu.addItem(.separator())

        startItem.target = self
        stopItem.target = self
        stopItem.isEnabled = false
        logItem.target = self
        menu.addItem(startItem)
        menu.addItem(stopItem)
        menu.addItem(logItem)

        menu.addItem(.separator())
        let quitItem = NSMenuItem(title: "Quit", action: #selector(quit), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    @objc private func refreshWindows() {
        windowsMenu.removeAllItems()
        let windows = listWindows()
        if windows.isEmpty {
            let item = NSMenuItem(title: "No capturable windows found", action: nil, keyEquivalent: "")
            item.isEnabled = false
            windowsMenu.addItem(item)
            return
        }

        for window in windows.prefix(40) {
            let item = NSMenuItem(title: window.label, action: #selector(selectWindow(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = window
            windowsMenu.addItem(item)
        }
    }

    @objc private func selectWindow(_ sender: NSMenuItem) {
        guard let window = sender.representedObject as? WatchWindow else {
            return
        }
        selectedWindow = window
        selectedItem.title = "Selected: \(window.label)"
        if watcher == nil {
            setStatus(.selected)
        }
    }

    @objc private func startWatching() {
        guard watcher == nil else {
            return
        }
        guard let selectedWindow else {
            selectedItem.title = "Select a window first"
            setStatus(.needsWindow)
            return
        }
        setStatus(.starting)

        let root = opswatchRoot()
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/env")
        process.currentDirectoryURL = root
        var arguments = [
            "go", "run", "./cmd/opswatch", "watch",
            "--vision-provider", env("OPSWATCH_VISION_PROVIDER", "ollama"),
            "--model", env("OPSWATCH_MODEL", "llama3.2-vision"),
            "--interval", env("OPSWATCH_INTERVAL", "10s"),
            "--window-id", "\(selectedWindow.id)",
            "--max-image-dimension", env("OPSWATCH_MAX_IMAGE_DIMENSION", "1000"),
            "--ollama-num-predict", env("OPSWATCH_OLLAMA_NUM_PREDICT", "128"),
            "--alert-cooldown", env("OPSWATCH_ALERT_COOLDOWN", "2m"),
            "--min-analysis-interval", env("OPSWATCH_MIN_ANALYSIS_INTERVAL", "30s"),
            "--environment", env("OPSWATCH_ENVIRONMENT", "prod"),
            "--notify",
            "--verbose"
        ]
        appendOptionalFlag("--intent", envOptional("OPSWATCH_INTENT"), to: &arguments)
        appendOptionalFlag("--expected-action", envOptional("OPSWATCH_EXPECTED_ACTION"), to: &arguments)
        appendOptionalFlag("--protected-domain", envOptional("OPSWATCH_PROTECTED_DOMAIN"), to: &arguments)
        process.arguments = arguments

        FileManager.default.createFile(atPath: logURL.path, contents: nil)
        do {
            logHandle = try FileHandle(forWritingTo: logURL)
            try logHandle?.seekToEnd()
        } catch {
            selectedItem.title = "Log error: \(error.localizedDescription)"
            return
        }
        process.standardOutput = logHandle
        process.standardError = logHandle
        process.terminationHandler = { [weak self] process in
            Task { @MainActor in
                guard let self else {
                    return
                }
                if self.watcher === process {
                    self.watcher = nil
                    self.startItem.isEnabled = true
                    self.stopItem.isEnabled = false
                    self.setStatus(process.terminationStatus == 0 ? .selected : .stoppedUnexpectedly)
                    try? self.logHandle?.close()
                    self.logHandle = nil
                }
            }
        }

        do {
            try process.run()
            watcher = process
            startItem.isEnabled = false
            stopItem.isEnabled = true
            setStatus(.watching)
            NSWorkspace.shared.open(logURL)
        } catch {
            selectedItem.title = "Start failed: \(error.localizedDescription)"
            setStatus(.error)
            try? logHandle?.close()
            logHandle = nil
        }
    }

    @objc private func stopWatching() {
        watcher?.terminate()
        watcher = nil
        startItem.isEnabled = true
        stopItem.isEnabled = false
        setStatus(selectedWindow == nil ? .idle : .selected)
        try? logHandle?.close()
        logHandle = nil
    }

    @objc private func openLog() {
        NSWorkspace.shared.open(logURL)
    }

    @objc private func quit() {
        stopWatching()
        NSApp.terminate(nil)
    }

    private func listWindows() -> [WatchWindow] {
        let options: CGWindowListOption = [.optionOnScreenOnly, .excludeDesktopElements]
        guard let rawWindows = CGWindowListCopyWindowInfo(options, kCGNullWindowID) as? [[String: Any]] else {
            return []
        }

        return rawWindows.compactMap { info in
            guard let id = info[kCGWindowNumber as String] as? UInt32,
                  let owner = info[kCGWindowOwnerName as String] as? String else {
                return nil
            }
            let layer = info[kCGWindowLayer as String] as? Int ?? 0
            let alpha = info[kCGWindowAlpha as String] as? Double ?? 1
            if layer != 0 || alpha <= 0 {
                return nil
            }
            let title = info[kCGWindowName as String] as? String ?? ""
            if owner == "OpsWatchBar" || owner == "Window Server" {
                return nil
            }
            return WatchWindow(id: id, owner: owner, title: title)
        }
    }

    private func env(_ key: String, _ fallback: String) -> String {
        let value = ProcessInfo.processInfo.environment[key] ?? ""
        return value.isEmpty ? fallback : value
    }

    private func envOptional(_ key: String) -> String? {
        let value = ProcessInfo.processInfo.environment[key] ?? ""
        return value.isEmpty ? nil : value
    }

    private func appendOptionalFlag(_ flag: String, _ value: String?, to arguments: inout [String]) {
        guard let value, !value.isEmpty else {
            return
        }
        arguments.append(flag)
        arguments.append(value)
    }

    private func setStatus(_ status: WatchStatus) {
        statusItem.button?.title = status.menuTitle
        statusItemRow.title = "Status: \(status.description)"
    }

    private func opswatchRoot() -> URL {
        if let value = ProcessInfo.processInfo.environment["OPSWATCH_ROOT"], !value.isEmpty {
            return URL(fileURLWithPath: value)
        }

        var url = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0..<5 {
            if FileManager.default.fileExists(atPath: url.appendingPathComponent("go.mod").path) {
                return url
            }
            url.deleteLastPathComponent()
        }
        return URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
    }
}

private enum WatchStatus {
    case idle
    case selected
    case needsWindow
    case starting
    case watching
    case stoppedUnexpectedly
    case error

    var menuTitle: String {
        switch self {
        case .idle:
            return "OpsWatch"
        case .selected:
            return "OpsWatch ◦"
        case .needsWindow:
            return "OpsWatch !"
        case .starting:
            return "OpsWatch …"
        case .watching:
            return "OpsWatch ●"
        case .stoppedUnexpectedly:
            return "OpsWatch !"
        case .error:
            return "OpsWatch !"
        }
    }

    var description: String {
        switch self {
        case .idle:
            return "idle"
        case .selected:
            return "window selected"
        case .needsWindow:
            return "select a window first"
        case .starting:
            return "starting watcher"
        case .watching:
            return "watching"
        case .stoppedUnexpectedly:
            return "watcher stopped"
        case .error:
            return "error"
        }
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
