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
   index fa6eed7e..4b173c5b 100644
   --- a/stubservice/main.go
   +++ b/stubservice/main.go
   @@ -85,6 +85,7 @@ func init() {
    	// Validate STORAGE_BACKEND
    	switch storageBackend {
    	case "gcs":
   +	case "mapstorage":
    	default:
    		logrus.Fatal("Invalid STORAGE_BACKEND value")
    	}
   @@ -200,6 +201,14 @@ func main() {

    			store := backends.NewGCS(gcsStorageClient, gcsBucket, time.Hour*24)
    			stubHandler = stubhandlers.NewRedirectHandler(store, cdnPrefix, gcsPrefix, bouncerBaseURL)
   +		} else if storageBackend == "mapstorage" {
   +			logrus.WithFields(logrus.Fields{
   +				"backend": storageBackend,
   +				"bucket":  "",
   +				"prefix":  "",
   +				"cdn":     cdnPrefix,
   +			}).Info("Starting in redirect mode")
   +			stubHandler = stubhandlers.NewRedirectHandler(backends.NewMapStorage(), cdnPrefix, "", bouncerBaseURL)
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

### How to use `pprof` locally?

Apply the following diff in addition to the previous one (about `mapstorage`):

```diff
diff --git a/stubservice/main.go b/stubservice/main.go
index fa6eed7e..d7e533b7 100644
--- a/stubservice/main.go
+++ b/stubservice/main.go
@@ -9,6 +9,7 @@ import (
 	"fmt"
 	"io/ioutil"
 	"net/http"
+	"net/http/pprof"
 	"net/url"
 	"os"
 	"strconv"
@@ -228,6 +229,11 @@ func main() {
 	mux.HandleFunc("/__heartbeat__", okHandler)
 	mux.HandleFunc("/__version__", versionHandler)
 	mux.HandleFunc("/__pingdom__", pingdomHandler)
+	mux.HandleFunc("/debug/pprof/", pprof.Index)
+	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
+	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
+	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
+	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

 	logrus.Fatal(http.ListenAndServe(addr, mux))
 }
```

Start the stub service:

```
RETURN_MODE=redirect \
STORAGE_BACKEND=mapstorage \
CDN_PREFIX=cdn-prefix/ \
HMAC_KEY=testkey \
BASE_URL=http://localhost:8000/ \
go run stubservice/main.go
```

#### CPU profile

Collect a profile:

```
curl -s http://127.0.0.1:8000/debug/pprof/profile > ./cpu.out
```

Load and read the collected profile:

```
go tool pprof -http=:8080 ./cpu.out
```
