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
}
