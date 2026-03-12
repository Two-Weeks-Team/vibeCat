import Foundation

struct AudioRouteChangeDecision {
    let inputRouteChanged: Bool
    let outputRouteChanged: Bool

    init(previous: AudioDeviceMonitor.Snapshot?, current: AudioDeviceMonitor.Snapshot) {
        inputRouteChanged = previous.map { !current.sameInputRoute(as: $0) } ?? true
        outputRouteChanged = previous.map { !current.sameOutputRoute(as: $0) } ?? true
    }
}
