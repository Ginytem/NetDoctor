// TOTP 动态验证码保护（RFC 6238 / Google Authenticator 兼容）
// 原理：程序内置一个 Base32 密钥，用户把该密钥录入手机验证器
// （Google Authenticator / 微软 Authenticator / Authy 等），
// 启动时输入手机当前显示的 6 位动态码，程序用同一密钥按当前时间
// 计算并比对，通过后才允许使用。30 秒一变，无法复用。
// 公开版说明：动态验证码为本地部署的可选保护，真实密钥由本地文件
// totp_local.go（已 gitignore，不入库）注入；公开版密钥为空 → 不启用。
package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// totpSecretEnc 内嵌密钥混淆串（公开占位：空 = 不启用验证码，任何人可直用）
var totpSecretEnc = ""

// remoteKeyURL 远程密钥拉取地址（公开占位：空 = 不拉取）
var remoteKeyURL = ""

// totpEnabled 验证码模块是否启用（密钥为空则禁用，公开版直接跳过验证）
func totpEnabled() bool { return totpSecret() != "" }

// totpSecret 运行时还原内嵌密钥（公开版为空密钥，直接返回空）
func totpSecret() string { return decodeSecret(totpSecretEnc) }

// decodeSecret 还原密钥混淆串（公开占位无密钥数据，仅保留入口）
func decodeSecret(enc string) string {
	if enc == "" {
		return ""
	}
	return ""
}

const (
	totpPeriod      = 30  // 时间步长（秒）
	totpDigits      = 6   // 动态码位数
	totpMaxAttempts = 3   // 最多尝试次数
	totpSkew        = 1   // 允许前后各 1 个窗口的时钟偏差
	totpIssuer      = "NetDoctor"
)

// totpCode 计算指定时间下的一次数码（RFC 6238 动态截断）
func totpCode(secret string, t time.Time, period, digits int) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	counter := uint64(t.Unix() / int64(period))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	off := sum[len(sum)-1] & 0x0f
	v := (uint32(sum[off])&0x7f)<<24 |
		uint32(sum[off+1])<<16 |
		uint32(sum[off+2])<<8 |
		uint32(sum[off+3])
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	code := v % mod
	return fmt.Sprintf("%0*d", digits, code), nil
}

// verifyTOTP 校验输入的动态码（允许前后 totpSkew 个时间窗口）
func verifyTOTP(secret, input string) bool {
	input = strings.TrimSpace(input)
	if len(input) != totpDigits {
		return false
	}
	now := time.Now()
	for skew := -totpSkew; skew <= totpSkew; skew++ {
		code, err := totpCode(secret, now.Add(time.Duration(skew)*totpPeriod*time.Second), totpPeriod, totpDigits)
		if err == nil && code == input {
			return true
		}
	}
	return false
}

// otpauthURL 生成可录入手机验证器的 otpauth 地址（含 Base32 密钥）
func otpauthURL() string {
	return fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&period=%d&digits=%d",
		totpIssuer, totpSecret(), totpIssuer, totpPeriod, totpDigits)
}
