import Foundation
import CoreGraphics

public enum ImageDiffer {
    private static let thumbnailSize = 32

    public static func hasSignificantChange(from previous: CGImage?, to current: CGImage, threshold: Double = 0.05) -> Bool {
        guard let previous else { return true }
        guard let prevThumb = thumbnail(previous), let currThumb = thumbnail(current) else { return true }
        let diff = pixelDiff(prevThumb, currThumb)
        return diff > threshold
    }

    private static func thumbnail(_ image: CGImage) -> [UInt8]? {
        let size = thumbnailSize
        var pixels = [UInt8](repeating: 0, count: size * size * 4)
        guard let context = CGContext(
            data: &pixels,
            width: size,
            height: size,
            bitsPerComponent: 8,
            bytesPerRow: size * 4,
            space: CGColorSpaceCreateDeviceRGB(),
            bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
        ) else { return nil }
        context.draw(image, in: CGRect(x: 0, y: 0, width: size, height: size))
        return pixels
    }

    private static func pixelDiff(_ a: [UInt8], _ b: [UInt8]) -> Double {
        guard a.count == b.count, !a.isEmpty else { return 1.0 }
        var total: Double = 0
        for i in stride(from: 0, to: a.count, by: 4) {
            let dr = Double(a[i]) - Double(b[i])
            let dg = Double(a[i+1]) - Double(b[i+1])
            let db = Double(a[i+2]) - Double(b[i+2])
            total += sqrt(dr*dr + dg*dg + db*db) / (255.0 * sqrt(3.0))
        }
        return total / Double(a.count / 4)
    }
}
