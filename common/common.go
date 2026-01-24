package common

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mojocn/base64Captcha"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"go.uber.org/zap"
)

const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
const DefaultExpiredToken = 24 * 30 * 3600
const DefaultCaptchaExpired = 5 * time.Minute

var DefaultDomain = "http://zlku49474541.vicp.fun"

var DefaultJwtKey = []byte("FiAVkmPYIgFY1osjyKRufkxkHgtCS1Ff")

const DefaultKeyIssuer = "jiayou"

var DefaultLogger *zap.Logger

var GlobalDevice = ""
var GlobalToken = ""

var XrayProcess *os.Process

var DefaultXrayConfig = "xray.json"
var DefaultXrayPath = "xray.exe"

var DefaultFFmpegPath = "ffmpeg.exe"

const DefaultXrayHost = "127.0.0.1"
const DefaultXrayPort = 9878
const DefaultXrayUser = "jiayou"
const DefaultXrayPwd = "jiayou"

var DefaultProxy = fmt.Sprintf("http://%s:%s@%s:%d", DefaultXrayUser, DefaultXrayPwd, DefaultXrayHost, DefaultXrayPort)

type Claims struct {
	UserID   uint   `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func ApiImagePath(file string) string {
	return DefaultDomain + "/api/static/images/" + file
}
func GenerateToken(userID uint, userName string) (string, error) {
	expirationTime := time.Now().Add(DefaultExpiredToken * time.Second)
	claims := &Claims{
		UserID:   userID,
		Username: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    DefaultKeyIssuer,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(DefaultJwtKey)
}
func VerifyToken(token string, outUser *string) bool {
	var claims Claims
	t, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return DefaultJwtKey, nil
	})
	*outUser = claims.Username
	return err == nil && t.Valid
}
func GenerateCaptcha() (string, string) {
	var conf = base64Captcha.ConfigCharacter{
		Height:            40,
		Width:             194,
		Mode:              base64Captcha.CaptchaModeNumberAlphabet,
		ComplexOfNoiseDot: 50,
		CaptchaLen:        4,
		//IsShowNoiseDot:    true,
		//IsShowHollowLine: true,
		//IsShowSlimeLine:  true,
		//IsShowSineLine:   true,
	}
	var id, engine = base64Captcha.GenerateCaptcha("", conf)
	return id, base64.StdEncoding.EncodeToString(engine.BinaryEncoding())
}
func GenerateVideoFirstFrame(file string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := ffmpeg.Input(file).
		Silent(true).
		Output("pipe:", ffmpeg.KwArgs{
			"ss":      "00:00:00",
			"vframes": 1,
			"vcodec":  "png",
			"f":       "image2pipe",
			"pix_fmt": "rgb24",
		}).
		SetFfmpegPath(DefaultFFmpegPath).
		WithOutput(buf).
		Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg run failed,%w", err)
	}
	return buf.Bytes(), nil

}
func GenerateCrc32(data any) (uint32, error) {
	if data == nil {
		return 0, nil
	}
	var reader io.Reader
	switch v := data.(type) {
	case []byte:
		reader = bytes.NewReader(v)
	case string:
		reader = strings.NewReader(v)
	case io.Reader:
		reader = v
	}
	hash := crc32.NewIEEE()
	_, err := io.Copy(hash, reader)
	if err != nil {
		return 0, err
	}
	return hash.Sum32(), nil
}
func Hmac256(message, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(message)
	return h.Sum(nil)
}
func Sha256(message []byte) []byte {
	hash := sha256.New()
	hash.Write(message)
	return hash.Sum(nil)
}
func PwdHash(password string) string {
	var raw = md5.Sum([]byte(password))
	return hex.EncodeToString(raw[:])
}
func GetFileSize(file string) (int64, error) {
	fp, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer fp.Close()
	return fp.Seek(0, io.SeekEnd)

}

func VerifyCaptcha(id, value string) bool {
	return base64Captcha.VerifyCaptcha(id, value)
}

func init() {
	base64Captcha.Expiration = DefaultCaptchaExpired
}
