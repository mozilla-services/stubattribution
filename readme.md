Stub Attribution
===

lambda function accepts an attribution code and returns a modified stub installer containing the attribution code.

Environment Variables
===

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
