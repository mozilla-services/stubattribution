package attributioncode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var validAttributionKeys = map[string]bool{
	"source":   true,
	"medium":   true,
	"campaign": true,
	"content":  true,
}

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
	if len(code) > 200 {
		return "", errors.New("code longer than 200 characters")
	}

	unEscapedCode, err := url.QueryUnescape(code)
	if err != nil {
		return "", errors.Wrap(err, "QueryUnescape")
	}
	vals, err := url.ParseQuery(unEscapedCode)
	if err != nil {
		return "", errors.Wrap(err, "ParseQuery")
	}

	if v.HMACKey != "" {
		if err := v.validateSignature(code, sig); err != nil {
			return "", err
		}
	}

	if err := v.validateTimestamp(vals.Get("timestamp")); err != nil {
		return "", err
	}
	vals.Del("timestamp")

	// all keys are valid
	for k := range vals {
		if !validAttributionKeys[k] {
			return "", errors.Errorf("%s is not a valid attribution key", k)
		}
	}

	// all keys are included
	if len(vals) != len(validAttributionKeys) {
		return "", errors.New("code is missing keys")
	}

	// source key in whitelist
	if source := vals.Get("source"); !isWhitelisted(source) {
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
