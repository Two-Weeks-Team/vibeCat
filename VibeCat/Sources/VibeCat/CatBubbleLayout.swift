import AppKit

struct CatBubblePlacement {
    let frame: NSRect
    let tailDirection: ChatBubbleView.TailDirection
    let tailRatio: CGFloat
}

enum CatBubbleLayout {
    private static let edgeInset: CGFloat = 8
    private static let bubbleGap: CGFloat = 8

    static func placement(
        catFrame: NSRect,
        bubbleSize: NSSize,
        screenFrame: NSRect,
        mode: ChatBubbleView.DisplayMode,
        reservedBottomMinY: CGFloat? = nil
    ) -> CatBubblePlacement {
        var bubbleX = catFrame.midX - bubbleSize.width / 2
        var bubbleY = catFrame.maxY + bubbleGap
        var tailDirection: ChatBubbleView.TailDirection = .bottom

        switch mode {
        case .status:
            let topAnchor = min(catFrame.minY - bubbleGap, (reservedBottomMinY ?? catFrame.minY) - bubbleGap)
            bubbleY = max(screenFrame.minY + edgeInset, topAnchor - bubbleSize.height)
            tailDirection = .top
        case .speech:
            let projectedTop = bubbleY + bubbleSize.height
            if projectedTop > screenFrame.maxY - edgeInset {
                let topAnchor = min(catFrame.minY - bubbleGap, (reservedBottomMinY ?? catFrame.minY) - bubbleGap)
                bubbleY = max(screenFrame.minY + edgeInset, topAnchor - bubbleSize.height)
                tailDirection = .top
            }
        }

        let minX = screenFrame.minX + edgeInset
        let maxX = screenFrame.maxX - bubbleSize.width - edgeInset
        bubbleX = min(max(bubbleX, minX), maxX)

        let tailLocalX = catFrame.midX - bubbleX
        let tailRatio = tailLocalX / bubbleSize.width

        return CatBubblePlacement(
            frame: NSRect(x: bubbleX, y: bubbleY, width: bubbleSize.width, height: bubbleSize.height),
            tailDirection: tailDirection,
            tailRatio: tailRatio
        )
    }
}
