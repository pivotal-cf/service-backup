#!/usr/bin/env bash
set -eu
set -o pipefail

server_hostname="s3.amazonaws.com"

set +e
sudo rm /etc/ssl/certs/ca-certificates.crt 2>/dev/null
set -e

sudo mkdir -p /etc/ssl/certs/

echo -n \
  | openssl s_client -connect "$server_hostname:443" 2>/dev/null \
  | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' \
  > ./ca-certificates.crt

sudo mv ./ca-certificates.crt /etc/ssl/certs/
