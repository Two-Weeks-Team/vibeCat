import CoreGraphics
import Foundation
import XCTest
@testable import VibeCatCore

final class ImageProcessorTests: XCTestCase {
    func testResizeIfNeededReturnsOriginalDimensionsWhenUnderLimit() {
        let image = makeSolidImage(width: 640, height: 480, color: (50, 120, 220, 255))
        let resized = ImageProcessor.resizeIfNeeded(image)

        XCTAssertEqual(resized.width, 640)
        XCTAssertEqual(resized.height, 480)
    }

    func testResizeIfNeededScalesDownLargeImagePreservingAspectRatio() {
        let image = makeSolidImage(width: 2048, height: 1024, color: (10, 20, 30, 255))
        let resized = ImageProcessor.resizeIfNeeded(image)

        XCTAssertEqual(resized.width, 1024)
        XCTAssertEqual(resized.height, 512)
    }

    func testToJPEGDataProducesJPEGHeader() {
        let image = makeSolidImage(width: 64, height: 64, color: (200, 150, 100, 255))
        let jpeg = ImageProcessor.toJPEGData(image)

        XCTAssertNotNil(jpeg)
        XCTAssertGreaterThan(jpeg?.count ?? 0, 4)
        XCTAssertEqual(jpeg?[0], 0xFF)
        XCTAssertEqual(jpeg?[1], 0xD8)
    }

    func testToBase64JPEGReturnsDecodableString() {
        let image = makeSolidImage(width: 64, height: 64, color: (90, 40, 10, 255))
        let base64 = ImageProcessor.toBase64JPEG(image)

        XCTAssertNotNil(base64)
        let decoded = base64.flatMap { Data(base64Encoded: $0) }
        XCTAssertNotNil(decoded)
        XCTAssertGreaterThan(decoded?.count ?? 0, 4)
    }

    private func makeSolidImage(width: Int, height: Int, color: (UInt8, UInt8, UInt8, UInt8)) -> CGImage {
        var pixels = [UInt8](repeating: 0, count: width * height * 4)
        for index in stride(from: 0, to: pixels.count, by: 4) {
            pixels[index] = color.0
            pixels[index + 1] = color.1
            pixels[index + 2] = color.2
            pixels[index + 3] = color.3
        }

        let context = CGContext(
            data: &pixels,
            width: width,
            height: height,
            bitsPerComponent: 8,
            bytesPerRow: width * 4,
            space: CGColorSpaceCreateDeviceRGB(),
            bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
        )!

        return context.makeImage()!
    }
}
