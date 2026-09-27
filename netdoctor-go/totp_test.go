package main

import (
	"testing"
	"time"
)

// RFC 6238 附录 B 官方测试向量（HMAC-SHA1，8 位码）
// secret = "12345678901234567890" 的 Base32 = GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ
func TestTotpRFC6238Vectors(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	cases := []struct {
		unix  int64
		code8 string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}
	for _, c := range cases {
		code, err := totpCode(secret, time.Unix(c.unix, 0), 30, 8)
		if err != nil {
			t.Fatalf("unix=%d err=%v", c.unix, err)
		}
		if code != c.code8 {
			t.Errorf("unix=%d: got %s want %s", c.unix, code, c.code8)
		}
	}
	// 6 位码：同一向量截取 %10^6
	code, _ := totpCode(secret, time.Unix(59, 0), 30, 6)
	if code != "287082" {
		t.Errorf("6-digit at unix=59: got %s want 287082", code)
	}
}

// 同一时刻计算应稳定，且窗口 ±1 内可通过
// 依赖真实密钥：本地版（totp_local.go 存在）执行；公开版（密钥为空）跳过
func TestVerifyTOTP(t *testing.T) {
	secret := totpSecret()
	if secret == "" {
		t.Skip("公开版无密钥，跳过本地密钥自检")
	}
	if secret != "<你的Base32密钥>" {
		t.Fatalf("secret mismatch: %q", secret)
	}
	code, err := totpCode(secret, time.Now(), totpPeriod, totpDigits)
	if err != nil || len(code) != totpDigits {
		t.Fatalf("compute current code failed: %v %q", err, code)
	}
	if !verifyTOTP(secret, code) {
		t.Error("current code should verify true")
	}
	if verifyTOTP(secret, "000000") {
		t.Error("wrong code should verify false")
	}
	if verifyTOTP(secret, "12345") {
		t.Error("short code should verify false")
	}
	if verifyTOTP(secret, "") {
		t.Error("empty code should verify false")
	}
}

// otpauth URL 结构正确（公开版无密钥时跳过）
func TestOtpauthURL(t *testing.T) {
	secret := totpSecret()
	if secret == "" {
		t.Skip("公开版无密钥，跳过 otpauth URL 自检")
	}
	u := otpauthURL()
	want := "otpauth://totp/NetDoctor?secret=" + secret + "&issuer=NetDoctor&period=30&digits=6"
	if u != want {
		t.Errorf("url mismatch:\n got %s\nwant %s", u, want)
	}
}
