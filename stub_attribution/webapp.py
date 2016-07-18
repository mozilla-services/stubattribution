import hashlib
import logging
import os.path
import sys
import urllib
import urlparse

from flask import Flask, request, abort, make_response, redirect
from stub_attribution.modify import write_attribution_data

import requests

app = Flask('stub_attribution')

sentry = None
if os.environ.get('SENTRY_DSN'):
    from raven.contrib.flask import Sentry
    sentry = Sentry(app, dsn=os.environ['SENTRY_DSN'])

BOUNCER_URL = os.environ.get('BOUNCER_URL', 'https://download.mozilla.org/')

# The supported RETURN_METHODs are 'direct' and 'redirect'
RETURN_METHOD = os.environ.get('RETURN_METHOD', 'direct')

# If RETURN_METHOD is redirect, S3_BUCKET must be set
if RETURN_METHOD == 'redirect':
    import boto3
    import botocore.exceptions

    S3_BUCKET = os.environ.get('S3_BUCKET')
    S3_PREFIX = os.environ.get('S3_PREFIX', '')
    CDN_PREFIX = os.environ.get('CDN_PREFIX',
                                'https://s3.amazonaws.com/%s/' % S3_BUCKET)

VALID_ATTRIBUTION_KEYS = set([
    'source',
    'medium',
    'campaign',
    'content',
    ])


def unique_key(download_url, attribution_code):
    """Return sha256 hash of download_url + '|' + attribution_code"""
    sha = hashlib.sha256()
    sha.update(download_url + "|" + attribution_code)
    return sha.hexdigest()


def s3_redirect_url(s3_object):
    return CDN_PREFIX + urllib.quote(s3_object.key)


def get_redirect(product, lang, os_, attribution_code):
    """Returns a redirect to a build with attribution_code"""
    params = {'product': product}
    if lang is not None:
        params['lang'] = lang
    if os_ is not None:
        params['os'] = os_

    r = requests.get(BOUNCER_URL, params=params, allow_redirects=False)
    if not r.is_redirect:
        abort(404)

    redirect_url = r.headers['Location']
    filename = urlparse.unquote(os.path.basename(redirect_url))

    s3 = boto3.resource('s3')
    s3_filename = (S3_PREFIX + 'builds/' +
                   product + '/' +
                   (lang if lang else 'default') + '/' +
                   (os_ if os_ else 'default') + '/' +
                   unique_key(redirect_url, attribution_code) + '/' +
                   filename)

    s3_object = s3.Object(S3_BUCKET, s3_filename)
    try:
        # Raises Exception if object doesn't exist
        s3_object.load()
        return redirect(s3_redirect_url(s3_object))
    except botocore.exceptions.ClientError:
        pass

    # Get content and write attribution_code
    r = requests.get(redirect_url)
    if r.status_code != 200:
        abort(404)

    stub = r.content
    if attribution_code:
        write_attribution_data(stub, attribution_code)

    s3_object.put(
        Body=stub,
        ContentType=r.headers.get('Content-Type', ''),
    )

    return redirect(s3_redirect_url(s3_object))


def get_direct(product, lang, os_, attribution_code):
    params = {'product': product}
    if lang is not None:
        params['lang'] = lang
    if os_ is not None:
        params['os'] = os_

    r = requests.get(BOUNCER_URL, params=params)
    if r.status_code != 200:
        abort(404)

    stub = r.content
    content_type = r.headers['Content-Type']
    filename = os.path.basename(r.url)

    # Write attribution_code to stub installer
    if attribution_code:
        write_attribution_data(stub, attribution_code)

    # Match content-type and filename
    resp = make_response(stub)
    resp.headers['Content-Type'] = content_type
    resp.headers['Content-Disposition'] = ('attachment; filename="%s"'
                                           % filename)
    return resp


class ValidationException(Exception):
    pass


def validate_attribution_code(code):
    if len(code) > 200:
        raise ValidationException("code longer than 200 characters")

    unquoted_code = urllib.unquote(code)
    code_parts = urlparse.parse_qsl(unquoted_code)
    if set([x[0] for x in code_parts]) != VALID_ATTRIBUTION_KEYS:
        raise ValidationException("code contains invalid or is missing keys")

    return urllib.urlencode(code_parts)


@app.route('/')
def stub_installer():
    """Returns a stub installer with an attribution_code

    Incoming request should contain the following parameters:
        * os
        * product
        * lang
        * attribution_code

    os, product, and lang are passed directly to bouncer.
    attribution_code is written to the returned binary.
    """

    def redirect_bouncer():
        params = {'product': request.args.get('product')}

        try:
            params['lang'] = request.args['lang']
        except KeyError:
            pass

        try:
            params['os'] = request.args['os']
        except KeyError:
            pass

        return redirect(BOUNCER_URL + "?" + urllib.urlencode(params))

    if not request.args.get('product'):
        abort(404)

    if not request.args.get('attribution_code'):
        return redirect_bouncer()

    try:
        attribution_code = validate_attribution_code(
                request.args['attribution_code'])
    except ValidationException:
        if sentry is not None:
            sentry.captureException()
        return redirect_bouncer()

    try:
        if RETURN_METHOD == 'redirect':
            return get_redirect(
                request.args['product'],
                request.args.get('lang'),
                request.args.get('os'),
                attribution_code,
            )
        else:
            return get_direct(
                request.args['product'],
                request.args.get('lang'),
                request.args.get('os'),
                attribution_code,
            )
    except Exception:
        if sentry is not None:
            sentry.captureException()
        return redirect_bouncer()


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
