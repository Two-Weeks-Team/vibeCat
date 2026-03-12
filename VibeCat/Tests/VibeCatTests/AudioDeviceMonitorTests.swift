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
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .defaultInputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )

        XCTAssertTrue(first.sameDevices(as: second))
    }

    func testSnapshotSameDevicesDetectsActualRouteChange() {
        let first = AudioDeviceMonitor.Snapshot(
            trigger: .deviceListChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .defaultOutputChanged,
            inputDeviceID: AudioObjectID(151),
            inputDeviceName: "AirPods Microphone",
            outputDeviceID: AudioObjectID(145),
            outputDeviceName: "AirPods Output",
            deviceInventoryIDs: [145, 151]
        )

        XCTAssertFalse(first.sameDevices(as: second))
    }

    func testSnapshotSameDevicesDetectsInventoryChangeWithSameDefaultRoutes() {
        let first = AudioDeviceMonitor.Snapshot(
            trigger: .deviceListChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .deviceListChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140, 145]
        )

        XCTAssertFalse(first.sameDevices(as: second))
        XCTAssertTrue(first.sameRoute(as: second))
    }

    func testSnapshotSameInputRouteDetectsOutputOnlyChange() {
        let first = AudioDeviceMonitor.Snapshot(
            trigger: .defaultOutputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .defaultOutputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(145),
            outputDeviceName: "Studio Display",
            deviceInventoryIDs: [140, 145]
        )

        XCTAssertTrue(first.sameInputRoute(as: second))
        XCTAssertFalse(first.sameOutputRoute(as: second))
    }

    func testSnapshotSameOutputRouteDetectsInputOnlyChange() {
        let first = AudioDeviceMonitor.Snapshot(
            trigger: .defaultInputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let second = AudioDeviceMonitor.Snapshot(
            trigger: .defaultInputChanged,
            inputDeviceID: AudioObjectID(151),
            inputDeviceName: "AirPods Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 151]
        )

        XCTAssertFalse(first.sameInputRoute(as: second))
        XCTAssertTrue(first.sameOutputRoute(as: second))
    }
}
