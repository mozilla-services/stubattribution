# stubtestclient

Returns a signed URL to be used against the stub attribution service.

## Usage

```
go get github.com/mozilla-services/stubattribution/stubtestclient
./stubtestclient -hmackey <shared-hmac-key>
```

### How to run the stub service locally?

Without a GCP/AWS developer account, we need to patch the stub service to use a
in-memory backend. Then, we can fully run the stub service locally and use the
test client to invoke it.

1. Apply this patch to enable a `mapstorage` storage backend:

   ```diff
   diff --git a/stubservice/main.go b/stubservice/main.go
   index 9acfc06..29a3c0a 100644
   --- a/stubservice/main.go
   +++ b/stubservice/main.go
   @@ -63,6 +63,7 @@ func init() {

       // Validate STORAGE_BACKEND
       switch storageBackend {
   +	case "mapstorage":
       case "gcs":
       default:
           logrus.Fatal("Invalid STORAGE_BACKEND value")
   @@ -167,6 +168,16 @@ func main() {

               store := backends.NewGCS(gcsStorageClient, gcsBucket, time.Hour*24)
               stubHandler = stubhandlers.NewRedirectHandler(store, cdnPrefix, gcsPrefix)
   +		} else if storageBackend == "mapstorage" {
   +			logrus.WithFields(logrus.Fields{
   +				"backend": storageBackend,
   +				"bucket":  "",
   +				"prefix":  "",
   +				"cdn":     cdnPrefix,
   +			}).Info("Starting in redirect mode")
   +
   +			store = backends.NewMapStorage()
   +			storagePrefix = ""
           } else {
               logrus.WithField("backend", storageBackend).Fatal("Unsupported storage backend")
           }
  ```

2. Run the service with the command line below:

   ```
   RETURN_MODE=redirect \
   STORAGE_BACKEND=mapstorage \
   CDN_PREFIX=cdn-prefix/ \
   HMAC_KEY=testkey \
   BASE_URL=http://localhost:8000/ \
   go run stubservice/main.go
   ```

### How to generate valid URLs to call the stub service?

Use a variation of this command line (which has sensitive default values to
verify the RTAMO feature):

```
go run testing/stubtestclient/main.go \
  -baseurl=http://localhost:8000 \
  -product=firefox-stub \
  -campaign=amo-fx-cta-123 \
  -content=rta:dUJsb2NrMEByYXltb25kaGlsbC5uZXQ \
  -source=addons.mozilla.org -medium=referral \
  -experiment="" \
  -variation=""
```

You can pass this command to `curl` directly, which is useful to pass extra
headers like a `Referer` header for RTAMO:

```
curl $(go run testing/stubtestclient/main.go -baseurl=http://localhost:8000 -product=firefox-stub -campaign=amo-fx-cta-123 -content=rta:dUJsb2NrMEByYXltb25kaGlsbC5uZXQ -source=addons.mozilla.org -medium=referral -experiment="" -variation="") -H 'Referer: https://www.mozilla.org/'
```
