package server

import (
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"

	adminv1 "admin/api/gen/admin/v1"
	authenticationv1 "admin/api/gen/authentication/v1"
)

const (
	captchaWidth       = 240
	captchaHeight      = 80
	captchaLength      = 4
	captchaDotCount    = 80
	captchaExpirySec   = 600
	captchaOpenCharset = "1234567890"
)

var captchaStore = base64Captcha.NewMemoryStore(1024, time.Duration(captchaExpirySec)*time.Second)

func newCaptchaDriver() *base64Captcha.DriverString {
	return base64Captcha.NewDriverString(
		captchaHeight,
		captchaWidth,
		captchaDotCount,
		base64Captcha.OptionShowHollowLine,
		captchaLength,
		captchaOpenCharset,
		nil,
		nil,
		[]string{"wqy-microhei.ttc"},
	)
}

func generateCaptcha() (*adminv1.GetCaptchaResponse, error) {
	driver := newCaptchaDriver()
	generated := base64Captcha.NewCaptcha(driver, captchaStore)

	id, b64s, _, err := generated.Generate()
	if err != nil {
		return nil, err
	}
	return &adminv1.GetCaptchaResponse{
		CaptchaId: id,
		ImageBase64: b64s,
		ExpiresIn: uint32(captchaExpirySec),
	}, nil
}

func verifyCaptcha(captchaID, code string) error {
	id := strings.TrimSpace(captchaID)
	answer := strings.TrimSpace(code)
	if id == "" || answer == "" {
		return authenticationv1.ErrorBadRequest("captcha_id and code are required")
	}

	if !captchaStore.Verify(id, answer, true) {
		return authenticationv1.ErrorUnauthorized("invalid captcha code")
	}
	return nil
}
