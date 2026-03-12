import CoreAudio
import XCTest
@testable import VibeCat

final class AudioRouteChangeDecisionTests: XCTestCase {
    func testInitialSnapshotTreatsBothRoutesAsChanged() {
        let current = AudioDeviceMonitor.Snapshot(
            trigger: .deviceListChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )

        let decision = AudioRouteChangeDecision(previous: nil, current: current)

        XCTAssertTrue(decision.inputRouteChanged)
        XCTAssertTrue(decision.outputRouteChanged)
    }

    func testOutputOnlyChangeRestartsOutputOnly() {
        let previous = AudioDeviceMonitor.Snapshot(
            trigger: .defaultOutputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let current = AudioDeviceMonitor.Snapshot(
            trigger: .defaultOutputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(145),
            outputDeviceName: "Studio Display",
            deviceInventoryIDs: [140, 145]
        )

        let decision = AudioRouteChangeDecision(previous: previous, current: current)

        XCTAssertFalse(decision.inputRouteChanged)
        XCTAssertTrue(decision.outputRouteChanged)
    }

    func testInputOnlyChangeRestartsInputOnly() {
        let previous = AudioDeviceMonitor.Snapshot(
            trigger: .defaultInputChanged,
            inputDeviceID: AudioObjectID(140),
            inputDeviceName: "MacBook Pro Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 140]
        )
        let current = AudioDeviceMonitor.Snapshot(
            trigger: .defaultInputChanged,
            inputDeviceID: AudioObjectID(151),
            inputDeviceName: "AirPods Microphone",
            outputDeviceID: AudioObjectID(133),
            outputDeviceName: "MacBook Pro Speakers",
            deviceInventoryIDs: [133, 151]
        )

        let decision = AudioRouteChangeDecision(previous: previous, current: current)

        XCTAssertTrue(decision.inputRouteChanged)
        XCTAssertFalse(decision.outputRouteChanged)
    }
}
