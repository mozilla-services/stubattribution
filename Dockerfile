FROM python:2.7

WORKDIR /app

COPY . /app

RUN pip install -r requirements.txt

EXPOSE 8000
CMD ["./bin/run"]
