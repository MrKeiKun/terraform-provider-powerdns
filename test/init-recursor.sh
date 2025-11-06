#!/bin/sh
set -eux

mkdir -p /etc/powerdns/api-config

# Ensure pdns can write
chown -R pdns:pdns /etc/powerdns/api-config || true
chmod -R 777 /etc/powerdns/api-config

exec pdns_recursor "$@"
