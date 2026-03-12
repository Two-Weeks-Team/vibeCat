import CoreAudio
import Foundation

@MainActor
final class AudioDeviceMonitor {
    enum Trigger: String, Sendable {
        case deviceListChanged = "device_list_changed"
        case defaultInputChanged = "default_input_changed"
        case defaultOutputChanged = "default_output_changed"
    }

    struct Snapshot: Equatable, Sendable {
        let trigger: Trigger
        let inputDeviceID: AudioObjectID
        let inputDeviceName: String
        let outputDeviceID: AudioObjectID
        let outputDeviceName: String

        func sameDevices(as other: Snapshot) -> Bool {
            inputDeviceID == other.inputDeviceID &&
                inputDeviceName == other.inputDeviceName &&
                outputDeviceID == other.outputDeviceID &&
                outputDeviceName == other.outputDeviceName
        }
    }

    var onChange: ((Snapshot) -> Void)?

    private let systemObjectID = AudioObjectID(kAudioObjectSystemObject)
    private let listenerQueue = DispatchQueue(label: "vibecat.audio-device-monitor")

    private var devicesAddress = AudioObjectPropertyAddress(
        mSelector: kAudioHardwarePropertyDevices,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    private var defaultInputAddress = AudioObjectPropertyAddress(
        mSelector: kAudioHardwarePropertyDefaultInputDevice,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    private var defaultOutputAddress = AudioObjectPropertyAddress(
        mSelector: kAudioHardwarePropertyDefaultOutputDevice,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )

    private var devicesListener: AudioObjectPropertyListenerBlock?
    private var defaultInputListener: AudioObjectPropertyListenerBlock?
    private var defaultOutputListener: AudioObjectPropertyListenerBlock?
    private var lastSnapshot: Snapshot?
    private var isStarted = false

    var latestSnapshot: Snapshot? {
        lastSnapshot
    }

    func start() {
        guard !isStarted else { return }
        isStarted = true
        lastSnapshot = makeSnapshot(trigger: .deviceListChanged)

        registerListener(for: &devicesAddress, storage: &devicesListener, trigger: .deviceListChanged)
        registerListener(for: &defaultInputAddress, storage: &defaultInputListener, trigger: .defaultInputChanged)
        registerListener(for: &defaultOutputAddress, storage: &defaultOutputListener, trigger: .defaultOutputChanged)

        if let lastSnapshot {
            NSLog(
                "[AUDIO-DEVICE] monitor started input=%@(%u) output=%@(%u)",
                lastSnapshot.inputDeviceName,
                lastSnapshot.inputDeviceID,
                lastSnapshot.outputDeviceName,
                lastSnapshot.outputDeviceID
            )
        }
    }

    func stop() {
        guard isStarted else { return }
        removeListener(for: &devicesAddress, storage: &devicesListener)
        removeListener(for: &defaultInputAddress, storage: &defaultInputListener)
        removeListener(for: &defaultOutputAddress, storage: &defaultOutputListener)
        isStarted = false
    }

    func currentSnapshot() -> Snapshot {
        makeSnapshot(trigger: .deviceListChanged)
    }

    private func emitChange(_ trigger: Trigger) {
        let snapshot = makeSnapshot(trigger: trigger)
        if let lastSnapshot, trigger != .deviceListChanged, snapshot.sameDevices(as: lastSnapshot) {
            NSLog("[AUDIO-DEVICE] duplicate change ignored trigger=%@", trigger.rawValue)
            return
        }

        lastSnapshot = snapshot
        NSLog(
            "[AUDIO-DEVICE] detected change trigger=%@ input=%@(%u) output=%@(%u)",
            trigger.rawValue,
            snapshot.inputDeviceName,
            snapshot.inputDeviceID,
            snapshot.outputDeviceName,
            snapshot.outputDeviceID
        )
        onChange?(snapshot)
    }

    private func registerListener(
        for address: inout AudioObjectPropertyAddress,
        storage: inout AudioObjectPropertyListenerBlock?,
        trigger: Trigger
    ) {
        let block: AudioObjectPropertyListenerBlock = { [weak self] _, _ in
            Task { @MainActor [weak self] in
                self?.emitChange(trigger)
            }
        }
        let status = AudioObjectAddPropertyListenerBlock(systemObjectID, &address, listenerQueue, block)
        if status == noErr {
            storage = block
        } else {
            NSLog("[AUDIO-DEVICE] failed to register listener trigger=%@ status=%d", trigger.rawValue, status)
        }
    }

    private func removeListener(
        for address: inout AudioObjectPropertyAddress,
        storage: inout AudioObjectPropertyListenerBlock?
    ) {
        guard let block = storage else { return }
        let status = AudioObjectRemovePropertyListenerBlock(systemObjectID, &address, listenerQueue, block)
        if status != noErr {
            NSLog("[AUDIO-DEVICE] failed to remove listener status=%d", status)
        }
        storage = nil
    }

    private func makeSnapshot(trigger: Trigger) -> Snapshot {
        let inputDeviceID = defaultDeviceID(for: kAudioHardwarePropertyDefaultInputDevice)
        let outputDeviceID = defaultDeviceID(for: kAudioHardwarePropertyDefaultOutputDevice)
        return Snapshot(
            trigger: trigger,
            inputDeviceID: inputDeviceID,
            inputDeviceName: deviceName(for: inputDeviceID),
            outputDeviceID: outputDeviceID,
            outputDeviceName: deviceName(for: outputDeviceID)
        )
    }

    private func defaultDeviceID(for selector: AudioObjectPropertySelector) -> AudioObjectID {
        var address = AudioObjectPropertyAddress(
            mSelector: selector,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        var deviceID = AudioObjectID(kAudioObjectUnknown)
        var size = UInt32(MemoryLayout<AudioObjectID>.size)
        let status = AudioObjectGetPropertyData(systemObjectID, &address, 0, nil, &size, &deviceID)
        if status != noErr {
            NSLog("[AUDIO-DEVICE] failed to resolve default device selector=%u status=%d", selector, status)
            return AudioObjectID(kAudioObjectUnknown)
        }
        return deviceID
    }

    private func deviceName(for deviceID: AudioObjectID) -> String {
        guard deviceID != AudioObjectID(kAudioObjectUnknown) else { return "Unknown" }

        var address = AudioObjectPropertyAddress(
            mSelector: kAudioObjectPropertyName,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        var size = UInt32(MemoryLayout<Unmanaged<CFString>?>.size)
        var unmanagedName: Unmanaged<CFString>?
        let status = withUnsafeMutablePointer(to: &unmanagedName) { pointer in
            AudioObjectGetPropertyData(deviceID, &address, 0, nil, &size, pointer)
        }
        if status != noErr {
            NSLog("[AUDIO-DEVICE] failed to resolve device name id=%u status=%d", deviceID, status)
            return "Device \(deviceID)"
        }
        return unmanagedName?.takeUnretainedValue() as String? ?? "Device \(deviceID)"
    }
}
