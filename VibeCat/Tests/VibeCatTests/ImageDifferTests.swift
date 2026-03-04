import CoreGraphics
import Foundation
import XCTest
@testable import VibeCatCore

final class ImageDifferTests: XCTestCase {
    func testHasSignificantChangeReturnsTrueWhenPreviousImageIsNil() {
        let current = makeSolidImage(width: 32, height: 32, color: (255, 255, 255, 255))
        XCTAssertTrue(ImageDiffer.hasSignificantChange(from: nil, to: current))
    }

    func testHasSignificantChangeReturnsFalseForIdenticalImages() {
        let image = makeSolidImage(width: 32, height: 32, color: (10, 40, 80, 255))
        XCTAssertFalse(ImageDiffer.hasSignificantChange(from: image, to: image))
    }

    func testHasSignificantChangeReturnsTrueForLargeVisualDifference() {
        let black = makeSolidImage(width: 32, height: 32, color: (0, 0, 0, 255))
        let white = makeSolidImage(width: 32, height: 32, color: (255, 255, 255, 255))

        XCTAssertTrue(ImageDiffer.hasSignificantChange(from: black, to: white, threshold: 0.05))
    }

    func testHasSignificantChangeRespectsThresholdForSmallDifference() {
        let basePixels = [UInt8](repeating: 0, count: 32 * 32 * 4)
        var changedPixels = basePixels
        changedPixels[0] = 255
        changedPixels[3] = 255

        let base = makeImage(width: 32, height: 32, pixels: basePixels)
        let changed = makeImage(width: 32, height: 32, pixels: changedPixels)

        XCTAssertTrue(ImageDiffer.hasSignificantChange(from: base, to: changed, threshold: 0.0001))
        XCTAssertFalse(ImageDiffer.hasSignificantChange(from: base, to: changed, threshold: 0.001))
    }

    private func makeSolidImage(width: Int, height: Int, color: (UInt8, UInt8, UInt8, UInt8)) -> CGImage {
        var pixels = [UInt8](repeating: 0, count: width * height * 4)
        for index in stride(from: 0, to: pixels.count, by: 4) {
            pixels[index] = color.0
            pixels[index + 1] = color.1
            pixels[index + 2] = color.2
            pixels[index + 3] = color.3
        }
        return makeImage(width: width, height: height, pixels: pixels)
    }

    private func makeImage(width: Int, height: Int, pixels: [UInt8]) -> CGImage {
        var mutablePixels = pixels
        let context = CGContext(
            data: &mutablePixels,
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
