import logging
import os.path
import sys

from flask import Flask, request, abort, make_response
from stub_attribution.modify import write_attribution_data

import requests

app = Flask('stub_attribution')


@app.route('/')
def stub_installer():
    if not request.args.get('product'):
        abort(404)

    try:
        params = {
            'os': request.args.get('os', ''),
            'lang': request.args.get('lang', ''),
            'product': request.args.get('product', ''),
        }
        r = requests.get('https://download.mozilla.org/', params=params)
    except:
        app.logger.exception('requests error:')
        abort(500)

    if r.status_code != 200:
        abort(404)
    stub = r.content
    content_type = r.headers['Content-Type']
    filename = os.path.basename(r.url)

    data = request.args.get('code', '')
    if data:
        try:
            write_attribution_data(stub, data)
        except:
            app.logger.exception('write_attribution_data error:')
            abort(400)
    resp = make_response(stub)
    resp.headers['Content-Type'] = content_type
    resp.headers['Content-Disposition'] = ('attachment; filename="%s"'
                                           % filename)
    return resp


@app.route('/__heartbeat__')
def heartbeat():
    return ("OK", 200, {"Content-Type": "text/plain"})


@app.route('/__lbheartbeat__')
def lbheartbeat():
    return ("OK", 200, {"Content-Type": "text/plain"})


if not app.debug:
    logging.basicConfig(stream=sys.stdout, level=logging.WARNING)

if __name__ == '__main__':
    app.run()
