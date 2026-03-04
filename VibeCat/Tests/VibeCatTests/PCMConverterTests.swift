import Foundation
import XCTest
@testable import VibeCatCore

final class PCMConverterTests: XCTestCase {
    func testInt16ToFloat32ConvertsKnownValues() {
        let samples: [Int16] = [Int16.min, -1, 0, 1, Int16.max]
        let converted = PCMConverter.int16ToFloat32(samples)

        XCTAssertEqual(converted[0], Float(Int16.min) / Float(Int16.max), accuracy: 0.000001)
        XCTAssertEqual(converted[1], -1.0 / Float(Int16.max), accuracy: 0.000001)
        XCTAssertEqual(converted[2], 0, accuracy: 0.000001)
        XCTAssertEqual(converted[3], 1.0 / Float(Int16.max), accuracy: 0.000001)
        XCTAssertEqual(converted[4], 1.0, accuracy: 0.000001)
    }

    func testFloat32ToInt16ClampsOutOfRangeValues() {
        let samples: [Float] = [-2.0, -1.0, -0.5, 0, 0.5, 1.0, 2.0]
        let converted = PCMConverter.float32ToInt16(samples)

        XCTAssertEqual(converted[0], Int16.min + 1)
        XCTAssertEqual(converted[1], Int16.min + 1)
        XCTAssertEqual(converted[2], Int16(-0.5 * Float(Int16.max)))
        XCTAssertEqual(converted[3], 0)
        XCTAssertEqual(converted[4], Int16(0.5 * Float(Int16.max)))
        XCTAssertEqual(converted[5], Int16.max)
        XCTAssertEqual(converted[6], Int16.max)
    }

    func testBytesRoundTripPreservesSamples() {
        let original: [Int16] = [0, 1, -1, 1234, -2345, Int16.max, Int16.min]
        let data = PCMConverter.int16ToBytes(original)
        let decoded = PCMConverter.bytesToInt16(data)
        XCTAssertEqual(decoded, original)
    }

    func testBytesToInt16IgnoresTrailingByte() {
        let data = Data([0x34, 0x12, 0xFF])
        let decoded = PCMConverter.bytesToInt16(data)
        XCTAssertEqual(decoded.count, 1)
        XCTAssertEqual(decoded[0], 0x1234)
    }
}
