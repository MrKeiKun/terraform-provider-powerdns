#!/bin/sh
# Init script for PowerDNS Auth to fix LMDB volume permissions

# Ensure /data directory has proper ownership for pdns user (for when running non-root)
# Also ensure it works when running as root
chown -R root:root /data
chmod 777 /data

# Start PowerDNS Auth server
exec /usr/local/sbin/pdns_server-startup