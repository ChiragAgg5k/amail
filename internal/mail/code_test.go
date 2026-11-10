package mail

import "testing"

func TestOneTimeCode(t *testing.T) {
	cases := []struct {
		name string
		text string
		code string
		ok   bool
	}{
		{"sentence", "Your verification code\nYour code is 482913. It expires in 10 minutes.", "482913", true},
		{"keyword line wins", "Order 55512 shipped\nYour OTP: 7731", "7731", true},
		{"passcode", "Use passcode 90210 to sign in", "90210", true},
		{"fallback first run", "Sign in to Example\nEnter 123456 to continue", "123456", true},
		{"skips date in keyword line", "Code sent on 2026-11-10: 845210", "845210", true},
		{"skips year in fallback", "Copyright 2026\nEnter 3141 now", "3141", true},
		{"year only on keyword line", "Your code is 1984", "1984", true},
		{"year only elsewhere", "Copyright 2026 Example", "", false},
		{"too short", "Your code is 123", "", false},
		{"too long", "Your code is 123456789", "", false},
		{"inside word", "Your code is ABC12345", "", false},
		{"time and phone", "Call 555-0134 at 10:30 for your verification", "", false},
		{"decimal amount", "Total 1234.56 paid", "", false},
		{"eight digits", "verification: 12345678", "12345678", true},
		{"case insensitive keyword", "VERIFICATION 4321 and 9876", "4321", true},
		{"empty", "", "", false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			code, ok := OneTimeCode(test.text)
			if code != test.code || ok != test.ok {
				t.Fatalf("OneTimeCode(%q) = %q, %v; want %q, %v", test.text, code, ok, test.code, test.ok)
			}
		})
	}
}
