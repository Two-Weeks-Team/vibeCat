import Foundation

public enum NavigatorVoiceCommandDetector {
    public static func shouldRoute(
        _ command: String,
        context: NavigatorContextPayload,
        hasPendingNavigatorPrompt: Bool
    ) -> Bool {
        let lowered = command.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !lowered.isEmpty else { return false }
        if hasPendingNavigatorPrompt {
            return true
        }

        var executeScore = keywordScore(lowered, keywords: [
            "apply", "do it", "run it", "rerun", "retry", "fix", "execute", "take care of",
            "type", "enter", "paste", "fill", "write", "click", "press", "focus", "select",
            "volume", "mute", "unmute", "quieter", "louder",
            "반영", "적용", "실행", "다시 돌려", "다시 실행", "수정", "해결", "처리해", "눌러", "클릭", "입력", "붙여넣", "써", "선택", "포커스",             "볼륨", "음량", "음소거", "소리", "쳐"
        ])
        var openScore = keywordScore(lowered, keywords: [
            "open", "go to", "take me", "bring me", "jump", "navigate", "show me",
            "열어", "이동", "데려가", "가보자", "보여", "가자"
        ])
        var findScore = keywordScore(lowered, keywords: [
            "find", "look up", "search", "docs", "official", "where is", "locate", "input field", "text field", "search box",
            "찾아", "검색", "공식 문서", "위치", "어디", "입력창", "검색창", "텍스트 필드"
        ])
        var analyzeScore = keywordScore(lowered, keywords: [
            "explain", "summarize", "what is", "why", "how", "tell me about",
            "설명", "요약", "왜", "어떻게", "뭐야", "알려줘"
        ])

        if containsAny(lowered, ["입력해", "입력해줘", "붙여넣어", "써줘", "type ", "enter ", "paste ", "fill "]) {
            executeScore = max(executeScore, 0.88)
        }
        if containsAny(lowered, [
            "입력해죠", "입력해 줘", "입력해죠.", "입력해 줘.",
            "넣어줘", "넣어 줘", "붙여줘", "붙여 줘", "적어줘", "적어 줘"
        ]) {
            executeScore = max(executeScore, 0.9)
        }
        if containsAny(lowered, ["입력해 주세요", "넣어주세요", "넣어 주세요", "붙여넣어 주세요", "적어 주세요"]) {
            executeScore = max(executeScore, 0.9)
        }
        if containsAny(lowered, ["쳐줘", "쳐 줘", "입력하자", "넣어보자", "써보자", "쳐봐"]) {
            executeScore = max(executeScore, 0.88)
        }
        if containsAny(lowered, ["volume", "mute", "unmute", "quieter", "louder", "볼륨", "음량", "음소거", "소리 줄", "소리 키"]) {
            executeScore = max(executeScore, 0.84)
        }
        if containsAny(lowered, ["열어줘", "데려가", "가보자"]) {
            openScore = max(openScore, 0.72)
        }
        if containsAny(lowered, ["찾아줘", "찾아봐", "공식 문서", "official docs"]) {
            findScore = max(findScore, 0.72)
        }
        if hasVisibleTextInput(context) &&
            containsAny(lowered, ["입력", "붙여넣", "써", "넣어", "붙여", "적어", "쳐", "type", "enter", "paste", "fill", "write"]) {
            executeScore = max(executeScore, 0.84)
        }
        if hasVisibleTextInput(context) && hasTextEntryPayload(command) {
            executeScore = max(executeScore, 0.86)
        }
        if hasVisibleTextInput(context) && containsAny(lowered, ["a부터 z", "a 부터 z", "a to z", "alphabet", "알파벳"]) {
            executeScore = max(executeScore, 0.82)
        }
        if containsAny(lowered, ["explain", "설명해", "알려줘", "what is", "why"]) &&
            executeScore < 0.4 && openScore < 0.4 && findScore < 0.4 {
            analyzeScore = max(analyzeScore, 0.72)
        }

        let ranked = [
            (NavigatorIntentClass.executeNow, executeScore),
            (NavigatorIntentClass.openOrNavigate, openScore),
            (NavigatorIntentClass.findOrLookup, findScore),
            (NavigatorIntentClass.analyzeOnly, analyzeScore),
        ].sorted { lhs, rhs in
            if lhs.1 == rhs.1 {
                return lhs.0.rawValue < rhs.0.rawValue
            }
            return lhs.1 > rhs.1
        }

        guard let top = ranked.first else { return false }
        let secondScore = ranked.dropFirst().first?.1 ?? 0
        guard top.1 >= 0.58 else { return false }
        guard top.0 != .analyzeOnly else { return false }
        guard top.1 - secondScore >= 0.12 else { return false }
        return true
    }

    private static func keywordScore(_ text: String, keywords: [String]) -> Double {
        var score = 0.0
        for keyword in keywords where text.contains(keyword) {
            score += 0.28
        }
        return min(score, 0.96)
    }

    private static func containsAny(_ text: String, _ keywords: [String]) -> Bool {
        keywords.contains { text.contains($0) }
    }

    private static func looksLikeTextInputRole(_ role: String) -> Bool {
        let lowered = role.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return containsAny(lowered, ["textfield", "textarea", "searchfield"])
    }

    private static func hasVisibleTextInput(_ context: NavigatorContextPayload) -> Bool {
        if looksLikeTextInputRole(context.focusedRole) {
            return true
        }
        if !context.inputFieldHint.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return true
        }
        if context.visibleInputCandidateCount > 0 {
            return true
        }
        let lowered = context.axSnapshot.lowercased()
        return containsAny(lowered, ["axtextfield", "axtextarea", "input:", "focused_input:"])
    }

    private static func referencesCurrentTarget(_ command: String) -> Bool {
        containsAny(command, ["여기", "거기", "here", "there", "current", "this"])
    }

    private static func hasTextEntryPayload(_ command: String) -> Bool {
        let lowered = command.lowercased()
        for marker in [
            "입력해줘:", "입력해:", "입력:", "입력해줘", "입력해죠", "입력해 줘",
            "붙여넣어:", "붙여넣기:", "넣어줘", "넣어 줘", "붙여줘", "붙여 줘", "적어줘", "적어 줘",
            "type:", "enter:", "paste:", "fill:"
        ] {
            guard let range = lowered.range(of: marker) else { continue }
            let remainder = lowered[range.upperBound...].trimmingCharacters(in: .whitespacesAndNewlines)
            if !remainder.isEmpty {
                return true
            }
        }

        // Postfix marker detection (e.g., "LGTM이라고 입력해줘")
        for suffix in [
            "이라고 입력해줘", "이라고 입력해", "이라고 입력",
            "라고 입력해줘", "라고 입력해", "라고 입력",
            "이라고 쳐줘", "라고 쳐줘", "이라고 써줘", "라고 써줘",
            "이라고 넣어줘", "라고 넣어줘", "이라고 적어줘", "라고 적어줘",
        ] {
            if let range = lowered.range(of: suffix) {
                let prefix = lowered[lowered.startIndex..<range.lowerBound]
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                if !prefix.isEmpty {
                    return true
                }
            }
        }

        for quote in ["\"", "'"] {
            let hasPairedQuotes = quote == "'" ? command.components(separatedBy: "'").count >= 3 : true
            if !hasPairedQuotes {
                continue
            }
            let parts = command.split(separator: Character(quote), omittingEmptySubsequences: true)
            if parts.count >= 2 {
                let candidate = parts[1].trimmingCharacters(in: .whitespacesAndNewlines)
                if !candidate.isEmpty {
                    return true
                }
            }
        }
        return false
    }
}
