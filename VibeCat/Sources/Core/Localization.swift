import Foundation

public enum AppLanguage: String, CaseIterable, Sendable {
    case ko
    case en
    case ja

    public static func resolve(_ rawValue: String?) -> AppLanguage {
        let normalized = rawValue?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        switch normalized {
        case "en", "eng", "english":
            return .en
        case "ja", "jp", "jpn", "japanese", "日本語":
            return .ja
        default:
            return .ko
        }
    }
}

public enum VibeCatL10n {
    private static func currentLanguage(_ language: String? = nil) -> AppLanguage {
        AppLanguage.resolve(language ?? AppSettings.shared.language)
    }

    private static func pick(_ language: AppLanguage, ko: String, en: String, ja: String) -> String {
        switch language {
        case .ko:
            return ko
        case .en:
            return en
        case .ja:
            return ja
        }
    }

    public static func languageDisplayName(_ code: String) -> String {
        switch AppLanguage.resolve(code) {
        case .ko:
            return "한국어"
        case .en:
            return "English"
        case .ja:
            return "日本語"
        }
    }

    public static func characterName(_ character: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch character {
        case "cat":
            return pick(lang, ko: "고양이", en: "Cat", ja: "猫")
        case "derpy":
            return pick(lang, ko: "더피", en: "Derpy", ja: "ダーピー")
        case "jinwoo":
            return pick(lang, ko: "진우", en: "Jinwoo", ja: "ジヌ")
        case "kimjongun":
            return pick(lang, ko: "김정운", en: "Kimjongun", ja: "キムジョンウン")
        case "saja":
            return pick(lang, ko: "사자", en: "Saja", ja: "サジャ")
        case "trump":
            return pick(lang, ko: "트럼프", en: "Trump", ja: "トランプ")
        default:
            return character
        }
    }

    public static func captureTargetModeTitle(_ mode: CaptureTargetMode, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch mode {
        case .windowUnderCursor:
            return pick(lang, ko: "커서 아래 창", en: "Window Under Cursor", ja: "カーソル下のウィンドウ")
        case .frontmostWindow:
            return pick(lang, ko: "전면 창", en: "Frontmost Window", ja: "最前面ウィンドウ")
        case .display:
            return pick(lang, ko: "디스플레이 전체", en: "Display", ja: "ディスプレイ全体")
        }
    }

    public static func chattinessOptionTitle(_ value: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch value {
        case "quiet":
            return pick(lang, ko: "조용하게", en: "Quiet", ja: "控えめ")
        case "chatty":
            return pick(lang, ko: "수다스럽게", en: "Chatty", ja: "おしゃべり")
        default:
            return pick(lang, ko: "보통", en: "Normal", ja: "標準")
        }
    }

    public static func speakerLabel(isUser: Bool, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        if isUser {
            return pick(lang, ko: "나", en: "You", ja: "あなた")
        }
        return pick(lang, ko: "AI", en: "AI", ja: "AI")
    }

    public static func toolDisplayName(_ raw: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "google_search", "search":
            return pick(lang, ko: "Google 검색", en: "Google Search", ja: "Google 検索")
        case "maps":
            return pick(lang, ko: "Google Maps", en: "Google Maps", ja: "Google Maps")
        case "url_context":
            return pick(lang, ko: "URL 컨텍스트", en: "URL Context", ja: "URL コンテキスト")
        case "code_execution":
            return pick(lang, ko: "코드 실행", en: "Code Execution", ja: "コード実行")
        case "file_search":
            return pick(lang, ko: "파일 검색", en: "File Search", ja: "ファイル検索")
        default:
            return raw
        }
    }

    public static func appMenuEditTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "편집", en: "Edit", ja: "編集")
    }

    public static func menuUndo(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "실행 취소", en: "Undo", ja: "取り消す")
    }

    public static func menuRedo(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "다시 실행", en: "Redo", ja: "やり直す")
    }

    public static func menuCut(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "잘라내기", en: "Cut", ja: "切り取り")
    }

    public static func menuCopy(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "복사", en: "Copy", ja: "コピー")
    }

    public static func menuPaste(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "붙여넣기", en: "Paste", ja: "貼り付け")
    }

    public static func menuSelectAll(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "전체 선택", en: "Select All", ja: "すべてを選択")
    }

    public static func menuLanguage(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "언어", en: "Language", ja: "言語")
    }

    public static func menuVoice(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "목소리", en: "Voice", ja: "音声")
    }

    public static func menuChattiness(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "말수", en: "Chattiness", ja: "会話量")
    }

    public static func menuCharacter(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "캐릭터", en: "Character", ja: "キャラクター")
    }

    public static func menuRecentSpeech(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "최근 대화", en: "Recent Speech", ja: "最近の会話")
    }

    public static func menuEmotionHistory(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "감정 변화", en: "Emotion History", ja: "感情履歴")
    }

    public static func menuModel(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "모델", en: "Model", ja: "モデル")
    }

    public static func menuLiveAPIModel(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Live API 모델", en: "Live API Model", ja: "Live API モデル")
    }

    public static func menuBackendAnalysisModels(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "백엔드 분석 모델", en: "Backend Analysis Models", ja: "バックエンド分析モデル")
    }

    public static func menuVisionSearch(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "비전/검색", en: "Vision/Search", ja: "ビジョン/検索")
    }

    public static func menuSupport(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "보조", en: "Support", ja: "補助")
    }

    public static func menuCaptureInterval(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "캡처 주기", en: "Capture Interval", ja: "キャプチャ間隔")
    }

    public static func menuCaptureTarget(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "캡처 대상", en: "Capture Target", ja: "キャプチャ対象")
    }

    public static func menuAdvanced(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "고급", en: "Advanced", ja: "詳細")
    }

    public static func menuPrivacy(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "개인정보", en: "Privacy", ja: "プライバシー")
    }

    public static func menuBackgroundMusic(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "배경 음악", en: "Background Music", ja: "BGM")
    }

    public static func menuGoogleSearch(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Google 검색", en: "Google Search", ja: "Google 検索")
    }

    public static func menuProactiveAudio(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "선제 음성", en: "Proactive Audio", ja: "プロアクティブ音声")
    }

    public static func menuManualAnalysisOnly(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "수동 분석만", en: "Manual Analyze Only", ja: "手動分析のみ")
    }

    public static func menuAnalyzeNow(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "지금 분석", en: "Analyze Now", ja: "今すぐ分析")
    }

    public static func menuNoScreenshotsStored(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "스크린샷 저장 안 함", en: "No screenshots saved", ja: "スクリーンショットは保存しません")
    }

    public static func menuConnect(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "연결...", en: "Connect...", ja: "接続...")
    }

    public static func menuReconnect(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "재연결", en: "Reconnect", ja: "再接続")
    }

    public static func menuPause(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "일시정지", en: "Pause", ja: "一時停止")
    }

    public static func menuResume(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "재개", en: "Resume", ja: "再開")
    }

    public static func menuMute(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "음소거", en: "Mute", ja: "ミュート")
    }

    public static func menuUnmute(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "음소거 해제", en: "Unmute", ja: "ミュート解除")
    }

    public static func menuQuit(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "VibeCat 종료", en: "Quit VibeCat", ja: "VibeCatを終了")
    }

    public static func statusConnected(latencyMs: Int, interactions: Int, sessionDuration: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return " 연결됨 · \(latencyMs)ms · 상호작용 \(interactions)회 · \(sessionDuration)"
        case .en:
            return " Connected · \(latencyMs)ms · \(interactions) interactions · \(sessionDuration)"
        case .ja:
            return " 接続中 · \(latencyMs)ms · \(interactions)回 · \(sessionDuration)"
        }
    }

    public static func statusReconnecting(attempt: Int, max: Int, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return " 재연결 중… (\(attempt)/\(max))"
        case .en:
            return " Reconnecting… (\(attempt)/\(max))"
        case .ja:
            return " 再接続中… (\(attempt)/\(max))"
        }
    }

    public static func statusDisconnected(lastSeen: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return " 연결 끊김 · 마지막 확인: \(lastSeen)"
        case .en:
            return " Disconnected · Last seen: \(lastSeen)"
        case .ja:
            return " 切断 · 最終確認: \(lastSeen)"
        }
    }

    public static func tooltipOfflineReconnecting(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "오프라인 · 재연결 중…", en: "Offline — reconnecting…", ja: "オフライン · 再接続中…")
    }

    public static func noRecentSpeech(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "최근 대화 없음", en: "No recent speech", ja: "最近の会話なし")
    }

    public static func noEmotionTransitions(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "감정 변화 없음", en: "No emotion transitions", ja: "感情変化なし")
    }

    public static func recentSpeechEntry(speaker: String, text: String, relative: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return "[\(speaker)] \(text) (\(relative))"
        case .en:
            return "[\(speaker)] \(text) (\(relative))"
        case .ja:
            return "[\(speaker)] \(text) (\(relative))"
        }
    }

    public static func duration(minutes: Int, seconds: Int, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return "\(minutes)분 \(seconds)초"
        case .en:
            return "\(minutes)m \(seconds)s"
        case .ja:
            return "\(minutes)分 \(seconds)秒"
        }
    }

    public static func relativeTime(from date: Date?, now: Date = Date(), language: String? = nil) -> String {
        let lang = currentLanguage(language)
        guard let date else {
            return pick(lang, ko: "없음", en: "never", ja: "なし")
        }

        let delta = Int(now.timeIntervalSince(date))
        if delta < 10 {
            return pick(lang, ko: "방금 전", en: "Just now", ja: "たった今")
        }
        if delta < 60 {
            return pick(lang, ko: "\(delta)초 전", en: "\(delta)s ago", ja: "\(delta)秒前")
        }
        if delta < 3600 {
            let minutes = delta / 60
            return pick(lang, ko: "\(minutes)분 전", en: "\(minutes)m ago", ja: "\(minutes)分前")
        }
        let hours = delta / 3600
        return pick(lang, ko: "\(hours)시간 전", en: "\(hours)h ago", ja: "\(hours)時間前")
    }

    public static func screenReadingTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "화면 읽는 중...", en: "Reading screen...", ja: "画面を読み取り中...")
    }

    public static func screenReadingDetail(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "현재 창 분석 중", en: "Analyzing current window", ja: "現在のウィンドウを分析中")
    }

    public static func listeningTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "듣는 중...", en: "Listening...", ja: "聞き取り中...")
    }

    public static func listeningDetail(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "말씀을 듣고 있어요", en: "Listening to you", ja: "話を聞いています")
    }

    public static func searchingTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "검색 중...", en: "Searching...", ja: "検索中...")
    }

    public static func groundingTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "근거 확인 중...", en: "Checking sources...", ja: "根拠を確認中...")
    }

    public static func toolRunningTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "도구 실행 중...", en: "Running tool...", ja: "ツール実行中...")
    }

    public static func responsePreparingTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "답변 정리 중...", en: "Preparing response...", ja: "回答を整理中...")
    }

    public static func reconnectingTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "다시 연결 중...", en: "Reconnecting...", ja: "再接続中...")
    }

    public static func liveSessionRecoveringDetail(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Live 세션 복구 중", en: "Restoring live session", ja: "Liveセッションを復旧中")
    }

    public static func sessionResumePreparingDetail(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "세션 재개 준비 중", en: "Preparing session resume", ja: "セッション再開を準備中")
    }

    public static func liveSessionReconnectingStatus(attempt: Int, max: Int, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return "Live 세션 재연결 중 (\(attempt)/\(max))"
        case .en:
            return "Live session reconnecting (\(attempt)/\(max))"
        case .ja:
            return "Liveセッション再接続中 (\(attempt)/\(max))"
        }
    }

    public static func characterChanging(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "캐릭터 변경 중...", en: "Switching character...", ja: "キャラクター切り替え中...")
    }

    public static func captureIndicatorLive(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "화면 캡처 켜짐", en: "Screen Capture On", ja: "画面キャプチャ ON")
    }

    public static func captureIndicatorManual(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "수동 분석 모드", en: "Manual Analyze Mode", ja: "手動分析モード")
    }

    public static func captureIndicatorPaused(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "캡처 일시정지", en: "Capture Paused", ja: "キャプチャ一時停止")
    }

    public static func captureIndicatorNoStorage(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "스크린샷 저장 안 함", en: "No screenshots saved", ja: "スクリーンショットは保存しません")
    }

    public static func characterAppeared(_ name: String, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return "\(name) 등장!"
        case .en:
            return "\(name) is here!"
        case .ja:
            return "\(name) 登場!"
        }
    }

    public static func sourceCount(_ count: Int, language: String? = nil) -> String {
        let lang = currentLanguage(language)
        switch lang {
        case .ko:
            return "근거 \(count)개"
        case .en:
            return "\(count) sources"
        case .ja:
            return "根拠 \(count)件"
        }
    }

    public static func toolStatusDetail(_ raw: String, language: String? = nil) -> String? {
        let lang = currentLanguage(language)
        switch raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "google_search", "search":
            return pick(lang, ko: "Google 검색 확인 중", en: "Checking Google Search", ja: "Google 検索を確認中")
        case "maps":
            return pick(lang, ko: "Google Maps 확인 중", en: "Checking Google Maps", ja: "Google Mapsを確認中")
        case "url_context":
            return pick(lang, ko: "URL 내용 읽는 중", en: "Reading URL content", ja: "URL内容を読み取り中")
        case "code_execution":
            return pick(lang, ko: "Code Execution 확인 중", en: "Checking Code Execution", ja: "Code Executionを確認中")
        case "file_search":
            return pick(lang, ko: "File Search 확인 중", en: "Checking File Search", ja: "File Searchを確認中")
        default:
            return nil
        }
    }

    public static func processingStateLabel(stage: String, tool: String = "", language: String? = nil) -> String {
        switch stage.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "searching":
            return searchingTitle(language: language)
        case "grounding":
            return groundingTitle(language: language)
        case "tool_running":
            return toolRunningTitle(language: language)
        case "response_preparing":
            return responsePreparingTitle(language: language)
        case "screen_analyzing":
            return screenReadingTitle(language: language)
        default:
            let name = toolDisplayName(tool, language: language).trimmingCharacters(in: .whitespacesAndNewlines)
            if !name.isEmpty {
                return toolRunningTitle(language: language)
            }
            return responsePreparingTitle(language: language)
        }
    }

    public static func processingStateDetail(stage: String, tool: String = "", sourceCount: Int? = nil, language: String? = nil) -> String? {
        let lang = currentLanguage(language)
        switch stage.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "searching":
            return toolStatusDetail("search", language: language)
        case "grounding":
            if let sourceCount {
                switch lang {
                case .ko:
                    return "Google 검색 · 근거 \(sourceCount)개 확인"
                case .en:
                    return "Google Search · checking \(sourceCount) sources"
                case .ja:
                    return "Google 検索 · 根拠 \(sourceCount)件を確認中"
                }
            }
            return toolStatusDetail("search", language: language)
        case "tool_running":
            return toolStatusDetail(tool, language: language)
        case "response_preparing":
            let name = toolDisplayName(tool, language: language).trimmingCharacters(in: .whitespacesAndNewlines)
            if name.isEmpty {
                return pick(lang, ko: "답변을 정리하는 중", en: "Preparing the reply", ja: "返答を整理中")
            }
            switch lang {
            case .ko:
                return "\(name) 결과 정리 중"
            case .en:
                return "Preparing \(name) results"
            case .ja:
                return "\(name) の結果を整理中"
            }
        case "screen_analyzing":
            return screenReadingDetail(language: language)
        default:
            return toolStatusDetail(tool, language: language)
        }
    }

    public static func onboardingWindowTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "VibeCat 연결", en: "VibeCat — Connect", ja: "VibeCat 接続")
    }

    public static func onboardingTitle(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "VibeCat에 연결", en: "Connect to VibeCat", ja: "VibeCat に接続")
    }

    public static func onboardingSubtitle(language: String? = nil) -> String {
        pick(
            currentLanguage(language),
            ko: "Gemini 호출은 VibeCat 백엔드를 통해 처리됩니다. 이 Mac에는 Gemini API 키를 저장하지 않습니다.",
            en: "Gemini calls run through the VibeCat backend. No Gemini API key is stored on this Mac.",
            ja: "Gemini 呼び出しは VibeCat バックエンド経由で実行されます。この Mac には Gemini API キーを保存しません。"
        )
    }

    public static func buttonConnect(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "연결", en: "Connect", ja: "接続")
    }

    public static func buttonCancel(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "취소", en: "Cancel", ja: "キャンセル")
    }

    public static func gatewayInvalidURL(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Gateway URL이 올바르지 않습니다. 설정을 확인하세요.", en: "Gateway URL invalid — check Settings", ja: "Gateway URL が無効です。設定を確認してください。")
    }

    public static func gatewayRegistrationFailedRetry(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Gateway 등록 실패 — 다시 연결해 주세요", en: "Gateway registration failed — retry connection", ja: "Gateway 登録に失敗しました。再接続してください")
    }

    public static func gatewayRegistrationFailedCheckBackend(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Gateway 등록 실패 — 백엔드 연결을 확인하세요", en: "Gateway registration failed — check backend connection", ja: "Gateway 登録に失敗しました。バックエンド接続を確認してください")
    }

    public static func gatewayRegistrationEmptySession(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "Gateway 등록 실패 — 세션 토큰이 비어 있습니다", en: "Gateway registration failed — empty session token", ja: "Gateway 登録に失敗しました。セッショントークンが空です")
    }

    public static func connectionTimeoutReconnecting(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "연결 시간이 초과되었습니다 — 다시 연결 중…", en: "Connection timed out — Reconnecting…", ja: "接続がタイムアウトしました — 再接続中…")
    }

    public static func connectionKeepsFailing(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "연결 실패가 반복됩니다 — gateway URL 또는 백엔드 상태를 확인하세요", en: "Connection keeps failing — check gateway URL or backend status", ja: "接続失敗が続いています — gateway URL またはバックエンド状態を確認してください")
    }

    public static func noInternetReconnecting(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "인터넷 연결이 없습니다 — 복구되면 다시 연결합니다", en: "No internet connection — will reconnect when available", ja: "インターネット接続がありません — 利用可能になり次第再接続します")
    }

    public static func connectionError(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "연결 오류", en: "Connection error", ja: "接続エラー")
    }

    public static func captureErrorNoDisplay(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "캡처할 디스플레이가 없습니다", en: "No display available for capture", ja: "キャプチャ可能なディスプレイがありません")
    }

    public static func decisionTrigger(_ value: String, language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "트리거: \(value)", en: "Trigger: \(value)", ja: "トリガー: \(value)")
    }

    public static func decisionVision(_ value: String, language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "비전: \(value)", en: "Vision: \(value)", ja: "ビジョン: \(value)")
    }

    public static func decisionMediator(_ value: String, language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "중재자: \(value)", en: "Mediator: \(value)", ja: "メディエーター: \(value)")
    }

    public static func decisionMood(_ value: String, language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "무드: \(value)", en: "Mood: \(value)", ja: "ムード: \(value)")
    }

    public static func decisionCooldown(_ value: String, language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "쿨다운: \(value)", en: "Cooldown: \(value)", ja: "クールダウン: \(value)")
    }

    public static func companionInputPlaceholder(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "메시지를 입력하고 Return을 누르세요", en: "Type a message and press Return", ja: "メッセージを入力して Return を押してください")
    }

    public static func companionListeningPrefix(language: String? = nil) -> String {
        pick(currentLanguage(language), ko: "듣는 중: ", en: "Listening: ", ja: "聞き取り中: ")
    }
}
