import CoreAudio
import XCTest
@testable import VibeCat

final class AudioDeviceMonitorTests: XCTestCase {
    func testSnapshotSameDevicesIgnoresTriggerDifference() {
        let first = AudioDeviceMonitor.Snapshot(
            trigger: .deviceListChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers"
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .defaultInputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers"
        )

        XCTAssertTrue(first.sameDevices(as: second))
    }

    func testSnapshotSameDevicesDetectsActualRouteChange() {
        let first = AudioDeviceMonitor.Snapshot(
            trigger: .deviceListChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers"
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .defaultOutputChanged,
            inputDeviceID: AudioObjectID(151),
            inputDeviceName: "AirPods Microphone",
            outputDeviceID: AudioObjectID(145),
            outputDeviceName: "AirPods Output"
        )

        XCTAssertFalse(first.sameDevices(as: second))
    }
}
