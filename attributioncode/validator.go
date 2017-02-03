package attributioncode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

var validAttributionKeys = map[string]bool{
	"source":   true,
	"medium":   true,
	"campaign": true,
	"content":  true,
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
	if len(code) > 200 {
		logEntry.WithField("code_len", len(code)).Error("code  longer than 200 characters")
		return "", errors.New("code longer than 200 characters")
	}

	unEscapedCode, err := base64Decoder.DecodeString(code)
	if err != nil {
		logEntry.WithError(err).Error("could not base64 decode code")
		return "", errors.Wrap(err, "DecodeString")
	}

	logEntry = logrus.WithField("code", unEscapedCode)

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

	if err := v.validateTimestamp(vals.Get("timestamp")); err != nil {
		logEntry.WithError(err).WithField("code_ts", vals.Get("timestamp")).Error("could not validate timestamp")
		return "", err
	}
	vals.Del("timestamp")

	// all keys are valid
	for k := range vals {
		if !validAttributionKeys[k] {
			logrus.WithField("invalid_key", k).Error("code contains invalidate key")
			return "", errors.Errorf("%s is not a valid attribution key", k)
		}
	}

	// all keys are included
	if len(vals) != len(validAttributionKeys) {
		logrus.Error("code is missing keys")
		return "", errors.New("code is missing keys")
	}

	// source key in whitelist
	if source := vals.Get("source"); !isWhitelisted(source) {
		logrus.WithField("source", source).Error("source is not in whitelist")
		return "", fmt.Errorf("source: %s is not in whitelist", source)
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

func (v *Validator) validateTimestamp(ts string) error {
	if ts == "" {
		return nil
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.Wrap(err, "Atoi")
	}

	if time.Since(time.Unix(tsInt, 0)) > v.Timeout {
		return errors.Errorf("Timestamp is older than timeout: %v", v.Timeout)
	}

	return nil
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
