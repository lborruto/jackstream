#!/usr/bin/env bash
# Re-fetches the bundled *.local-ip.medicmobile.org cert. Run manually before
# releasing a new image if the prior cert is near expiry (LE rotates every ~60d).
set -euo pipefail
cd "$(dirname "$0")/.."
curl -sfL https://local-ip.medicmobile.org/fullchain -o certs/fullchain.pem
curl -sfL https://local-ip.medicmobile.org/key -o certs/key.pem
chmod 644 certs/fullchain.pem
chmod 600 certs/key.pem
openssl x509 -in certs/fullchain.pem -noout -subject -dates
echo "certs refreshed."
