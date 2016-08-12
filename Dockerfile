FROM golang:1.6

ENV PROJECT=github.com/mozilla-services/go-stub_attribution

COPY version.json /app/version.json
COPY . /go/src/$PROJECT

ENV ADDR=8000
EXPOSE 8000

RUN go install $PROJECT/stubservice

CMD ['stubservice']
