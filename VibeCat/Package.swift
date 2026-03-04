// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "VibeCat",
    platforms: [.macOS(.v15)],
    targets: [
        .target(
            name: "VibeCatCore",
            path: "Sources/Core"
        ),
        .executableTarget(
            name: "VibeCat",
            dependencies: ["VibeCatCore"],
            path: "Sources/VibeCat"
        ),
        .testTarget(
            name: "VibeCatTests",
            dependencies: ["VibeCatCore"],
            path: "Tests/VibeCatTests"
        )
    ]
)
