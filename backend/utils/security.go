package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	defaultIssuer      = "MyApp"
	backupCodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	backupCodeLength   = 8
	defaultTotpPeriod  = 30
	totpValidationSkew = 1
	qrImageSize        = 256
)

func getEncryptionKey() ([]byte, error) {
	key := os.Getenv("TWOFA_ENCRYPTION_KEY")
	if key == "" {
		key = os.Getenv("JWT_SECRET")
	}
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("TWOFA_ENCRYPTION_KEY or JWT_SECRET must be set")
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:], nil
}

func EncryptSecret(plaintext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptSecret(encoded string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("invalid ciphertext")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func randomBase32Secret(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	return encoding.EncodeToString(raw), nil
}

func issuerName() string {
	if v := strings.TrimSpace(os.Getenv("APP_NAME")); v != "" {
		return v
	}
	return defaultIssuer
}

func GenerateKey(username, email string) (*otp.Key, string, error) {
	secret, err := randomBase32Secret(20)
	if err != nil {
		return nil, "", err
	}

	issuer := issuerName()
	identity := email
	if strings.TrimSpace(identity) == "" {
		identity = username
	}

	label := fmt.Sprintf("%s:%s", issuer, identity)
	otpauthURL := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s",
		url.PathEscape(label), secret, url.QueryEscape(issuer))

	key, err := otp.NewKeyFromURL(otpauthURL)
	if err != nil {
		return nil, "", err
	}
	return key, secret, nil
}

func GenerateQRCodeBase64(key *otp.Key) (string, error) {
	img, err := key.Image(qrImageSize, qrImageSize)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func ValidateTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) == 0 {
		return false
	}
	opts := totp.ValidateOpts{
		Period:    defaultTotpPeriod,
		Skew:      totpValidationSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	valid, err := totp.ValidateCustom(code, secret, now, opts)
	if err != nil {
		return false
	}
	return valid
}

func GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("invalid count")
	}
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		buf := make([]byte, backupCodeLength)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		var builder strings.Builder
		for _, b := range buf {
			builder.WriteByte(backupCodeAlphabet[int(b)%len(backupCodeAlphabet)])
		}
		codes[i] = builder.String()
	}
	return codes, nil
}
