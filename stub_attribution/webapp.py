from flask import Flask, request, abort
from os import getenv
from stub_attribution.modify import write_attribution_data
import boto3

S3_BUCKET = getenv('S3_BUCKET', '')
s3 = boto3.resource('s3')

app = Flask('stub_attribution')

@app.route('/<filepath>')
def stub_installer(filepath):
    try:
        key = s3.Object(S3_BUCKET, filepath)
        stub = bytearray(key.get()['Body'].read())
    except:
        abort(404)
    data = request.query_string
    if data:
        try:
            write_attribution_data(stub, data)
        except:
            abort(400)
    return stub

if __name__ == '__main__':
    app.run()
