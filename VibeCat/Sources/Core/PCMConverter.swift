import Foundation

public enum PCMConverter {
    public static func int16ToFloat32(_ samples: [Int16]) -> [Float] {
        samples.map { Float($0) / Float(Int16.max) }
    }

    public static func float32ToInt16(_ samples: [Float]) -> [Int16] {
        samples.map { sample in
            let clamped = max(-1.0, min(1.0, sample))
            return Int16(clamped * Float(Int16.max))
        }
    }

    public static func bytesToInt16(_ data: Data) -> [Int16] {
        var result = [Int16](repeating: 0, count: data.count / 2)
        _ = result.withUnsafeMutableBytes { ptr in
            data.copyBytes(to: ptr)
        }
        return result
    }

    public static func int16ToBytes(_ samples: [Int16]) -> Data {
        samples.withUnsafeBytes { Data($0) }
    }
}
