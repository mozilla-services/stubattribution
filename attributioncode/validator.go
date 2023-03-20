package attributioncode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// pre-compile regex
var (
	mozillaOrg = regexp.MustCompile(`^https://www.mozilla.org/`)
	rtamo      = regexp.MustCompile(`^rta:`)
)

// Set to match https://searchfox.org/mozilla-central/rev/a92ed79b0bc746159fc31af1586adbfa9e45e264/browser/components/attribution/AttributionCode.jsm#24
const maxUnescapedCodeLen = 1010

const downloadTokenField = "dltoken"

var validAttributionKeys = map[string]bool{
	"source":         true,
	"medium":         true,
	"campaign":       true,
	"content":        true,
	"experiment":     true,
	"installer_type": true,
	"variation":      true,
	"ua":             true,
	"visit_id":       true, // https://bugzilla.mozilla.org/show_bug.cgi?id=1677497
	"session_id":     true, // https://bugzilla.mozilla.org/show_bug.cgi?id=1809120
	"client_id":      true, // Alias of `visit_id`.
	"dlsource":       true, // https://github.com/mozilla-services/stubattribution/issues/159
}

// If any of these are not set in the incoming payload, they will be set to '(not set)'
var requiredAttributionKeys = []string{
	"source",
	"medium",
	"campaign",
	"content",
}

// These are not written to the installer.
var excludedAttributionKeys = []string{
	"visit_id",
	"session_id",
	"client_id",
}

var base64Decoder = base64.URLEncoding.WithPadding('.')

func generateDownloadToken() string {
	return uuid.NewString()
}

// Code represents a valid attribution code
type Code struct {
	Source        string
	Medium        string
	Campaign      string
	Content       string
	Experiment    string
	InstallerType string
	Variation     string
	UA            string
	ClientID      string
	SessionID     string
	DownloadSource      string

	downloadToken string

	rawURLVals url.Values
}

// DownloadToken returns unique token for this download.
func (c *Code) DownloadToken() string {
	if c.downloadToken == "" {
		c.downloadToken = generateDownloadToken()
	}

	return c.downloadToken
}

// URLEncode returns a query escaped stub attribution code
func (c *Code) URLEncode() string {
	for _, val := range excludedAttributionKeys {
		c.rawURLVals.Del(val)
	}
	c.rawURLVals.Set(downloadTokenField, c.DownloadToken())
	return url.QueryEscape(c.rawURLVals.Encode())
}

// FromRTAMO returns true when the content parameter contains a prefix for
// RTAMO, and false otherwise.
func (c *Code) FromRTAMO() bool {
	return rtamo.MatchString(c.Content)
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
func (v *Validator) Validate(code, sig, refererHeader string) (*Code, error) {
	logEntry := logrus.WithField("b64code", code)

	if code == "" {
		logEntry.Error("code is empty")
		return nil, errors.New("code is empty")
	}

	if len(code) > 5000 {
		logEntry.WithField("code_len", len(code)).Error("code longer than 5000 characters")
		return nil, errors.New("base64 code longer than 5000 characters")
	}

	if len(sig) > 5000 {
		logEntry.WithField("sig_len", len(sig)).Error("sig longer than 5000 characters")
		return nil, errors.New("sig longer than 5000 characters")
	}

	unEscapedCode, err := base64Decoder.DecodeString(code)
	if err != nil {
		logEntry.WithError(err).Error("could not base64 decode code")
		return nil, errors.Wrap(err, "DecodeString")
	}

	logEntry = logrus.WithField("code", unEscapedCode)
	if len(unEscapedCode) > maxUnescapedCodeLen {
		errMsg := fmt.Sprintf("code longer than %d characters", maxUnescapedCodeLen)
		logEntry.WithField("code_len", len(code)).Error(errMsg)
		return nil, errors.New(errMsg)
	}

	vals, err := url.ParseQuery(string(unEscapedCode))
	if err != nil {
		logEntry.WithError(err).Error("could not parse code")
		return nil, errors.Wrap(err, "ParseQuery")
	}

	if v.HMACKey != "" {
		if err := v.validateSignature(code, sig); err != nil {
			logEntry.WithError(err).Error("could not validate signature")
			return nil, err
		}
	}

	vals.Del("timestamp")

	// all keys are valid
	for k := range vals {
		if !validAttributionKeys[k] {
			logEntry.WithField("invalid_key", k).Error("code contains invalid key")
			return nil, errors.Errorf("%s is not a valid attribution key", k)
		}
	}

	if source := vals.Get("source"); !isWhitelisted(source) {
		logEntry.WithField("source", source).Error("source is not in whitelist")
		vals.Set("source", "(other)")
	}

	for _, val := range requiredAttributionKeys {
		if vals.Get(val) == "" {
			vals.Set(val, "(not set)")
		}
	}

	// The `visit_id` field is in fact the Google Analytics "client" ID and
	// "visit" ID seems confusing. Let's accept `client_id` as an alias of
	// `visit_id` so that Bedrock can start to send the value using the
	// `client_id` key instead of `visit_id`. We still allow the latter for
	// backward compatibility purposes.
	clientID := vals.Get("client_id")
	if clientID == "" {
		clientID = vals.Get("visit_id")
	}

	attributionCode := &Code{
		Source:        vals.Get("source"),
		Medium:        vals.Get("medium"),
		Campaign:      vals.Get("campaign"),
		Content:       vals.Get("content"),
		Experiment:    vals.Get("experiment"),
		InstallerType: vals.Get("installer_type"),
		Variation:     vals.Get("variation"),
		UA:            vals.Get("ua"),
		ClientID:      clientID,
		SessionID:     vals.Get("session_id"),
		DlSource:      vals.Get("dlsource"),

		rawURLVals: vals,
	}

	if attributionCode.FromRTAMO() {
		refererMatch := mozillaOrg.MatchString(refererHeader)

		if !refererMatch {
			logEntry.WithField("referer", refererHeader).Error("RTAMO attribution does not have https://www.mozilla.org referer header")
			return nil, errors.New("RTAMO attribution does not have https://www.mozilla.org referer header")
		}
	}

	return attributionCode, nil
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
