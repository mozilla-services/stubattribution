FROM golang:1.12

ENV PROJECT=github.com/mozilla-services/stubattribution

COPY . /app

ENV ADDR=":8000"
EXPOSE 8000

RUN cd /app && go install -mod vendor $PROJECT/stubservice

CMD ["stubservice"]
