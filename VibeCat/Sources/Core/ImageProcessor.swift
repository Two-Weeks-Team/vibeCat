import Foundation
import CoreGraphics
import ImageIO

public enum ImageProcessor {
    public static let maxDimension: CGFloat = 1024
    public static let jpegQuality: CGFloat = 0.7

    public static func resizeIfNeeded(_ image: CGImage) -> CGImage {
        let width = CGFloat(image.width)
        let height = CGFloat(image.height)
        guard width > maxDimension || height > maxDimension else { return image }

        let scale = maxDimension / max(width, height)
        let newWidth = Int(width * scale)
        let newHeight = Int(height * scale)

        guard let context = CGContext(
            data: nil,
            width: newWidth,
            height: newHeight,
            bitsPerComponent: 8,
            bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceRGB(),
            bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
        ) else { return image }

        context.draw(image, in: CGRect(x: 0, y: 0, width: newWidth, height: newHeight))
        return context.makeImage() ?? image
    }

    public static func toJPEGData(_ image: CGImage, quality: CGFloat = jpegQuality) -> Data? {
        let mutableData = NSMutableData()
        guard let destination = CGImageDestinationCreateWithData(
            mutableData, "public.jpeg" as CFString, 1, nil
        ) else { return nil }

        let options: [CFString: Any] = [
            kCGImageDestinationLossyCompressionQuality: quality
        ]
        CGImageDestinationAddImage(destination, image, options as CFDictionary)
        guard CGImageDestinationFinalize(destination) else { return nil }
        return mutableData as Data
    }

    public static func toBase64JPEG(_ image: CGImage, quality: CGFloat = jpegQuality) -> String? {
        guard let data = toJPEGData(image, quality: quality) else { return nil }
        return data.base64EncodedString()
    }
}
