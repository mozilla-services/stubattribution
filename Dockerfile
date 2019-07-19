FROM golang:1.12

ENV PROJECT=github.com/mozilla-services/stubattribution

COPY version.json /app/version.json
COPY . /go/src/$PROJECT

ENV ADDR=":8000"
EXPOSE 8000

RUN go install -mod vendor $PROJECT/stubservice

CMD ["stubservice"]
