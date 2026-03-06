import Foundation
import CoreGraphics
import ImageIO

public enum ImageProcessor {
    public static let jpegQuality: CGFloat = 0.65
    public static let maxDataSize = 3_145_728
    public static let maxPixelDimension = 2048

    public static func resizeIfNeeded(_ image: CGImage) -> CGImage {
        return image
    }

    public static func toJPEGData(_ image: CGImage, quality: CGFloat = jpegQuality) -> Data? {
        let prepared = prepareForEncoding(image)
        if let data = encodeJPEG(prepared, quality: quality), data.count <= maxDataSize {
            return data
        }
        return encodeJPEG(prepared, quality: 0.4)
    }

    public static func toBase64JPEG(_ image: CGImage, quality: CGFloat = jpegQuality) -> String? {
        guard let data = toJPEGData(image, quality: quality) else { return nil }
        return data.base64EncodedString()
    }

    private static func prepareForEncoding(_ image: CGImage) -> CGImage {
        let maxDim = max(image.width, image.height)
        let needsDownscale = maxDim > maxPixelDimension
        let targetWidth: Int
        let targetHeight: Int

        if needsDownscale {
            let scale = CGFloat(maxPixelDimension) / CGFloat(maxDim)
            targetWidth = Int(CGFloat(image.width) * scale)
            targetHeight = Int(CGFloat(image.height) * scale)
        } else {
            targetWidth = image.width
            targetHeight = image.height
        }

        let sRGB = CGColorSpace(name: CGColorSpace.sRGB) ?? CGColorSpaceCreateDeviceRGB()

        guard let context = CGContext(
            data: nil,
            width: targetWidth,
            height: targetHeight,
            bitsPerComponent: 8,
            bytesPerRow: targetWidth * 4,
            space: sRGB,
            bitmapInfo: CGImageAlphaInfo.noneSkipLast.rawValue
        ) else {
            return image
        }

        context.setFillColor(CGColor(red: 1, green: 1, blue: 1, alpha: 1))
        context.fill(CGRect(x: 0, y: 0, width: targetWidth, height: targetHeight))
        context.interpolationQuality = .high
        context.draw(image, in: CGRect(x: 0, y: 0, width: targetWidth, height: targetHeight))

        return context.makeImage() ?? image
    }

    private static func encodeJPEG(_ image: CGImage, quality: CGFloat) -> Data? {
        let mutableData = NSMutableData()
        guard let destination = CGImageDestinationCreateWithData(
            mutableData, "public.jpeg" as CFString, 1, nil
        ) else { return nil }
        let options: [CFString: Any] = [
            kCGImageDestinationLossyCompressionQuality: quality,
            kCGImageDestinationBackgroundColor: CGColor(red: 1, green: 1, blue: 1, alpha: 1),
        ]
        CGImageDestinationAddImage(destination, image, options as CFDictionary)
        guard CGImageDestinationFinalize(destination) else { return nil }
        return mutableData as Data
    }
}
