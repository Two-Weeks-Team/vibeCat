import XCTest
@testable import VibeCat

final class AssistantTranscriptionAssemblerTests: XCTestCase {
    func testLateChunkWithinMergeWindowExtendsSameUtterance() {
        var assembler = AssistantTranscriptionAssembler(mergeWindow: 0.75)
        let start = Date(timeIntervalSince1970: 1_700_000_000)

        XCTAssertEqual(
            assembler.ingest("정말이네요! 바이브캣이 크게 개선된 것", now: start),
            "정말이네요! 바이브캣이 크게 개선된 것"
        )
        XCTAssertTrue(assembler.markBoundary(now: start))

        XCTAssertEqual(
            assembler.ingest(" 같아 기뻐요,", now: start.addingTimeInterval(0.2)),
            "정말이네요! 바이브캣이 크게 개선된 것 같아 기뻐요,"
        )
        XCTAssertNil(assembler.finalizeIfDue(now: start.addingTimeInterval(0.8)))
        XCTAssertEqual(
            assembler.finalizeIfDue(now: start.addingTimeInterval(1.0)),
            "정말이네요! 바이브캣이 크게 개선된 것 같아 기뻐요,"
        )
    }

    func testChunkAfterMergeWindowStartsNewUtterance() {
        var assembler = AssistantTranscriptionAssembler(mergeWindow: 0.5)
        let start = Date(timeIntervalSince1970: 1_700_000_100)

        _ = assembler.ingest("첫 문장", now: start)
        XCTAssertTrue(assembler.markBoundary(now: start))

        XCTAssertEqual(
            assembler.ingest("다음 문장", now: start.addingTimeInterval(0.6)),
            "다음 문장"
        )
        XCTAssertFalse(assembler.hasPendingFinalization)
    }

    func testDiscardClearsBufferedUtterance() {
        var assembler = AssistantTranscriptionAssembler(mergeWindow: 0.5)

        _ = assembler.ingest("중간 조각")
        XCTAssertTrue(assembler.markBoundary())

        assembler.discard()

        XCTAssertNil(assembler.finalizeNow())
        XCTAssertEqual(assembler.currentText, "")
        XCTAssertFalse(assembler.hasPendingFinalization)
    }

    func testCumulativePartialReplacesExistingTranscriptInsteadOfDuplicating() {
        var assembler = AssistantTranscriptionAssembler(mergeWindow: 0.75)
        let start = Date(timeIntervalSince1970: 1_700_001_000)

        XCTAssertEqual(
            assembler.ingest("지금 Codex가 에디터", now: start),
            "지금 Codex가 에디터"
        )
        XCTAssertEqual(
            assembler.ingest("지금 Codex가 에디터에 열려 있어.", now: start.addingTimeInterval(0.1)),
            "지금 Codex가 에디터에 열려 있어."
        )
    }

    func testOverlappingPartialAppendsOnlyNewSuffix() {
        var assembler = AssistantTranscriptionAssembler(mergeWindow: 0.75)
        let start = Date(timeIntervalSince1970: 1_700_001_100)

        XCTAssertEqual(
            assembler.ingest("방금 바꾼 함수 하나", now: start),
            "방금 바꾼 함수 하나"
        )
        XCTAssertEqual(
            assembler.ingest("하나나 깨진 파일 하나부터 좁혀보자.", now: start.addingTimeInterval(0.1)),
            "방금 바꾼 함수 하나나 깨진 파일 하나부터 좁혀보자."
        )
    }
}
