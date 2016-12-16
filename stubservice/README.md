Stub Service
===

Accepts an attribution code and bouncer parameters and returns a, potentially, modified stub installer containing an attribution code.

Environment Variables
===

## HMAC_KEY
If set, the `attribution_code` parameter will be verified by validating that the
`attribution_sig` parameter matches the hex encoded sha256 hmac of `attribution_code` using
`HMAC_KEY`.

## HMAC_KEY_TIMEOUT (Default 10 minutes)
Will validate that the timestamp included in `attribution_code` is within (Now-timeout) to Now.

## SENTRY_DSN
If set, tracebacks will be sent to [Sentry](https://getsentry.com/).

## BOUNCER_URL (default: https://download.mozilla.org/)
Bouncer root URL.

## RETURN_METHOD
Can be 'direct' or 'redirect'.
### direct mode
Returns bytes directly to client
### redirect mode
Writes bytes to s3 and returns a redirect to S3 location.

## S3_BUCKET (redirect mode)
The bucket where builds will be written.

## S3_PREFIX (redirect mode)
A path prefix within the `S3_BUCKET` where builds will be written.

Default: ''

## CDN_PREFIX (redirect mode)
A prefix which will be added to the s3 key.

Default: 'https://s3.amazonaws.com/%s/' % S3_BUCKET
