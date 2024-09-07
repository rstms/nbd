#!/usr/bin/env python3

import requests
import click
import sys
from pathlib import Path
import base64
import json

class Netboot:
    def __init__(self, url, ca=None, cert=None, key=None):
        self.url = url
        self.kwargs = {}
        if ca:
            self.kwargs.update(dict(verify=ca))
        if cert and key:
            self.kwargs.update(dict(cert=(cert, key)))

    def encode_to_base64(self, input_string):
        input_bytes = input_string.encode('utf-8')
        base64_bytes = base64.b64encode(input_bytes)
        base64_string = base64_bytes.decode('utf-8')
        return base64_string 
    
    def parse_response(self, response):
        try:
            out = response.json()
        except json.JSONDecodeError:
            out = response.text.strip()
        return dict(
            status=response.status_code,
            result=out
        )
    
    def upload_package(self, package_file):
        with open(package_file, "rb") as fp:
            files = dict(uploadFile=(package_file.name, fp, "application/gzip"))
            response = requests.post(ctx.url + "/tarball/", files=files, **self.kwargs)
        return self.parse_response(response)

    def add(self, mac, os, response_file, package_file=None):
        if package_file:
            upload_result = self.upload_package(package_file)
        else:
            upload_result = None
        config=dict(
            address=mac,
            os=os,
            version='',
            config=self.encode_to_base64(response_file.read_text())
        )
        response = requests.put(self.url + '/host/', json=config, **self.kwargs)
        result = self.parse_response(response)
        result['package_upload'] = upload_result
        return result

    def ls(self):
        response = requests.get(self.url + "/hosts/", **self.kwargs)
        return self.parse_response(response)

    def delete_all(self):
        list_result = self.ls()
        macs = list_result['result']['addresses']
        results=[]
        for mac in macs:
            result = self.delete(mac)
            results.append(result)
        return results

    def delete(self, mac):
        response = requests.delete(self.url + "/host/", json=dict(address=mac), **self.kwargs)
        return self.parse_response(response)


@click.group
@click.option('-u', '--url', default='https://netboot.rstms.net/api', envvar='NBCTL_URL')
@click.option('-C', '--ca', default='/etc/ssl/keymaster.pem', envvar='NBCTL_CA')
@click.option('-c', '--cert', default='/etc/ssl/netboot.pem', envvar='NBCTL_CERT')
@click.option('-k', '--key', default='/etc/ssl/netboot.key', envvar='NBCTL_KEY')
@click.pass_context
def nbctl(ctx, url, ca, cert, key):
    """netboot configuration utility"""
    ctx.obj = Netboot(url, ca, cert, key)


@nbctl.command()
@click.argument('mac')
@click.argument('os')
@click.argument('response-file', type=click.Path(dir_okay=False, readable=True, path_type=Path))
@click.argument('package-file', required=False, type=click.Path(dir_okay=False, readable=True, path_type=Path))
@click.pass_obj
def add(ctx, mac, os, response_file, package_file):
    """add host config"""
    result = ctx.add(mac, os, response_file, package_file)
    click.echo(json.dumps(result, indent=2))



@nbctl.command()
@click.argument('package-file', type=click.Path(dir_okay=False, readable=True, path_type=Path))
@click.pass_obj
def upload(ctx, package_file):
    """upload package tarball"""
    result = ctx.upload_package(package_file)
    click.echo(json.dumps(result, indent=2))


@nbctl.command
@click.pass_obj
def ls(ctx):
    """list configs"""
    result = ctx.ls()
    click.echo(json.dumps(result, indent=2))

@nbctl.command
@click.argument('mac')
@click.pass_obj
def rm(ctx, mac):
    """delete host config"""
    if mac=='all':
        result = ctx.delete_all()
    else:
        result = ctx.delete(mac)
    click.echo(json.dumps(result, indent=2))


if __name__ == '__main__':
    sys.exit(nbctl())




