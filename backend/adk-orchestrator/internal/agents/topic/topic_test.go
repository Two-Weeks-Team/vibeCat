package topic

import (
	"reflect"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "detects keywords case-insensitively and de-duplicates",
			text: "BUG bug crash API api",
			want: []string{"bug", "crash", "api"},
		},
		{
			name: "returns keywords in configured order",
			text: "swift error",
			want: []string{"error", "swift"},
		},
		{
			name: "no keyword match",
			text: "hello world",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.text)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
