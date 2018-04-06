FROM golang:1.10

ENV PROJECT=github.com/mozilla-services/stubattribution

COPY version.json /app/version.json
COPY . /go/src/$PROJECT

ENV ADDR=":8000"
EXPOSE 8000

RUN go install $PROJECT/stubservice

CMD ["stubservice"]
