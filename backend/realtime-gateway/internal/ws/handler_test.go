package ws

import "testing"

func TestCouldBeQuestion(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"오늘 뉴스 한번 검색해줄래?", true},
		{"뉴스 검색해줘", true},
		{"최신 뉴스 알려줘?", true},
		{"날씨 어때?", true},
		{"오늘 날씨 알아봐줘", true},
		{"환율 찾아봐", true},
		{"search for today's news", true},
		{"오늘 뉴스 뭐야?", true},
		{"삼성전자 시가총액이 얼마야?", true},
		{"what time is it?", true},
		{"how does this work?", true},

		{"네", false},
		{"응", false},
		{"ㅋㅋ", false},
		{"hi", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := couldBeQuestion(tt.text)
			if got != tt.want {
				t.Errorf("couldBeQuestion(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}
