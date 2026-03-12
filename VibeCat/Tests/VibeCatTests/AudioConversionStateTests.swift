import AVFoundation
import XCTest
@testable import VibeCat

final class AudioConversionStateTests: XCTestCase {
    func testBeginTransitionInvalidatesPreviousGeneration() {
        let state = AudioConversionState()

        let initial = state.snapshot()
        XCTAssertTrue(state.isCurrentGeneration(initial.generation))
        XCTAssertFalse(initial.isTransitioning)

        state.beginDeviceTransition()

        let transitioned = state.snapshot()
        XCTAssertTrue(transitioned.isTransitioning)
        XCTAssertFalse(state.isCurrentGeneration(initial.generation))
        XCTAssertTrue(state.isCurrentGeneration(transitioned.generation))
    }

    func testConfigureConverterClearsTransitionAndPublishesConverter() {
        let state = AudioConversionState()
        let input = AVAudioFormat(commonFormat: .pcmFormatFloat32, sampleRate: 48_000, channels: 1, interleaved: false)

        state.beginDeviceTransition()
        let converter = state.configureConverter(for: input!)
        let snapshot = state.snapshot()

        XCTAssertNotNil(converter)
        XCTAssertNotNil(snapshot.converter)
        XCTAssertFalse(snapshot.isTransitioning)
    }

    func testClearConverterPreservesTransitionStateUntilRestartCompletes() {
        let state = AudioConversionState()
        let input = AVAudioFormat(commonFormat: .pcmFormatFloat32, sampleRate: 44_100, channels: 1, interleaved: false)

        _ = state.configureConverter(for: input!)
        state.beginDeviceTransition()
        state.clearConverter()

        let snapshot = state.snapshot()
        XCTAssertNil(snapshot.converter)
        XCTAssertTrue(snapshot.isTransitioning)

        state.finishDeviceTransition()
        XCTAssertFalse(state.snapshot().isTransitioning)
    }

    func testConfigureConverterReturnsNilForUnsupportedInputFormat() {
        let state = AudioConversionState()
        let input = AVAudioFormat(commonFormat: .pcmFormatFloat32, sampleRate: 44_100, channels: 0, interleaved: false)

        let converter = state.configureConverter(for: input!)

        XCTAssertNil(converter)
        XCTAssertNil(state.snapshot().converter)
    }
}
