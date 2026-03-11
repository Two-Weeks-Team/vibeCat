import AppKit
import Foundation
import VibeCatCore

/// FD kept open for process lifetime — OS releases the flock() on exit/crash.
nonisolated(unsafe) private var singleInstanceFD: Int32 = -1

@main
struct VibeCatApp {
    static func main() {
        let lockPath = NSTemporaryDirectory() + "com.vibecat.app.lock"
        singleInstanceFD = open(lockPath, O_CREAT | O_RDWR, 0o600)
        if singleInstanceFD == -1 || flock(singleInstanceFD, LOCK_EX | LOCK_NB) != 0 {
            NSLog("[VibeCat] Already running — exiting duplicate instance (lock: %@)", lockPath)
            if singleInstanceFD != -1 { close(singleInstanceFD) }
            return
        }

        let app = NSApplication.shared
        let delegate = AppDelegate()
        app.delegate = delegate
        installEditMenu()
        app.run()
    }

    /// Accessory apps have no menu bar, so Cmd+V (paste:) has nowhere to route.
    /// Install a hidden Edit menu so standard text editing shortcuts work in all text fields.
    @MainActor private static func installEditMenu() {
        let mainMenu = NSMenu()

        let editMenuItem = NSMenuItem()
        let editMenu = NSMenu(title: VibeCatL10n.appMenuEditTitle())
        editMenu.addItem(withTitle: VibeCatL10n.menuUndo(), action: Selector(("undo:")), keyEquivalent: "z")
        editMenu.addItem(withTitle: VibeCatL10n.menuRedo(), action: Selector(("redo:")), keyEquivalent: "Z")
        editMenu.addItem(.separator())
        editMenu.addItem(withTitle: VibeCatL10n.menuCut(), action: #selector(NSText.cut(_:)), keyEquivalent: "x")
        editMenu.addItem(withTitle: VibeCatL10n.menuCopy(), action: #selector(NSText.copy(_:)), keyEquivalent: "c")
        editMenu.addItem(withTitle: VibeCatL10n.menuPaste(), action: #selector(NSText.paste(_:)), keyEquivalent: "v")
        editMenu.addItem(withTitle: VibeCatL10n.menuSelectAll(), action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
        editMenuItem.submenu = editMenu

        mainMenu.addItem(editMenuItem)
        NSApplication.shared.mainMenu = mainMenu
    }
}
