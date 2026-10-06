package credential

import "testing"

func TestIsWeakAdminPassword(t *testing.T) {
	cases := []struct {
		name     string
		password string
		want     bool
	}{
		{"empty is unavailable", "", true},
		{"whitespace only", "   ", true},
		{"known default admin123", "admin123", true},
		{"case insensitive", "ADMIN123", true},
		{"known changeme", "changeme", true},
		{"long enough but not weak", "S3cure-Operator-Pass", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsWeakAdminPassword(tc.password); got != tc.want {
				t.Fatalf("IsWeakAdminPassword(%q) = %v, want %v", tc.password, got, tc.want)
			}
		})
	}
}

func TestIsShortAdminPassword(t *testing.T) {
	if !IsShortAdminPassword("") {
		t.Fatal("empty password must count as short")
	}
	// 长度门禁必须独立于弱口令清单：这个值不在清单里，但只有 11 字符。
	if !IsShortAdminPassword("NotWeakPw") {
		t.Fatal("short password that is not on the weak list must still be rejected")
	}
	short := make([]byte, MinAdminPasswordLength-1)
	for i := range short {
		short[i] = 'x'
	}
	exact := append(short, 'y')
	if !IsShortAdminPassword(string(short)) {
		t.Fatalf("%d-character password must be rejected (min = %d)", len(short), MinAdminPasswordLength)
	}
	if IsShortAdminPassword(string(exact)) {
		t.Fatalf("%d-character password must be accepted (min = %d)", len(exact), MinAdminPasswordLength)
	}
}
