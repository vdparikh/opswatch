// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "OpsWatchBar",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "OpsWatchBar", targets: ["OpsWatchBar"])
    ],
    targets: [
        .executableTarget(name: "OpsWatchBar")
    ]
)
