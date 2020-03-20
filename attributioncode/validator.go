package attributioncode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Set to match https://searchfox.org/mozilla-central/rev/a92ed79b0bc746159fc31af1586adbfa9e45e264/browser/components/attribution/AttributionCode.jsm#24
const maxUnescapedCodeLen = 1010

var validAttributionKeys = map[string]bool{
	"source":     true,
	"medium":     true,
	"campaign":   true,
	"content":    true,
	"experiment": true,
	"variation":  true,
	"ua":         true,
}

// If any of these are not set in the incoming payload, they will be set to '(not set)'
var requiredAttributionKeys = []string{
	"source",
	"medium",
	"campaign",
	"content",
}

var base64Decoder = base64.URLEncoding.WithPadding('.')

// Validator validates and returns santized attribution codes
type Validator struct {
	HMACKey string
	Timeout time.Duration
}

// NewValidator returns a new attribution code validator
func NewValidator(hmacKey string, timeout time.Duration) *Validator {
	return &Validator{
		HMACKey: hmacKey,
		Timeout: timeout,
	}
}

// Validate validates and sanitizes attribution code and signature
func (v *Validator) Validate(code, sig string) (string, error) {
	logEntry := logrus.WithField("b64code", code)

	if code == "" {
		logEntry.Error("code is empty")
		return "", errors.New("code is empty")
	}

	if len(code) > 5000 {
		logEntry.WithField("code_len", len(code)).Error("code longer than 5000 characters")
		return "", errors.New("base64 code longer than 5000 characters")
	}

	if len(sig) > 5000 {
		logEntry.WithField("sig_len", len(sig)).Error("sig longer than 5000 characters")
		return "", errors.New("sig longer than 5000 characters")
	}

	unEscapedCode, err := base64Decoder.DecodeString(code)
	if err != nil {
		logEntry.WithError(err).Error("could not base64 decode code")
		return "", errors.Wrap(err, "DecodeString")
	}

	logEntry = logrus.WithField("code", unEscapedCode)
	if len(unEscapedCode) > maxUnescapedCodeLen {
		errMsg := fmt.Sprintf("code longer than %d characters", maxUnescapedCodeLen)
		logEntry.WithField("code_len", len(code)).Error(errMsg)
		return "", errors.New(errMsg)
	}

	vals, err := url.ParseQuery(string(unEscapedCode))
	if err != nil {
		logEntry.WithError(err).Error("could not parse code")
		return "", errors.Wrap(err, "ParseQuery")
	}

	if v.HMACKey != "" {
		if err := v.validateSignature(code, sig); err != nil {
			logEntry.WithError(err).Error("could not validate signature")
			return "", err
		}
	}

	vals.Del("timestamp")

	// all keys are valid
	for k := range vals {
		if !validAttributionKeys[k] {
			logrus.WithField("invalid_key", k).Error("code contains invalidate key")
			return "", errors.Errorf("%s is not a valid attribution key", k)
		}
	}

	if source := vals.Get("source"); !isWhitelisted(source) {
		logrus.WithField("source", source).Error("source is not in whitelist")
		vals.Set("source", "(other)")
	}

	for _, val := range requiredAttributionKeys {
		if vals.Get(val) == "" {
			vals.Set(val, "(not set)")
		}
	}

	return url.QueryEscape(vals.Encode()), nil
}

func (v *Validator) validateSignature(code, sig string) error {
	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		return errors.Wrapf(err, "hex.DecodeString: %s", sig)
	}

	return checkMAC([]byte(v.HMACKey), []byte(code), sigBytes)
}

func checkMAC(key, msg, msgMAC []byte) error {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	expectedMac := mac.Sum(nil)
	if !hmac.Equal(msgMAC, expectedMac) {
		return errors.Errorf("HMAC would not validate. given: %x expected: %x", msgMAC, expectedMac)
	}
	return nil
}
