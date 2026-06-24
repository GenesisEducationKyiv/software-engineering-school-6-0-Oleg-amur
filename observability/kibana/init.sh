#!/bin/sh
set -eu

kibana_url="${KIBANA_URL:-http://kibana:5601}"
saved_objects_file="/usr/local/share/kibana/subscription-service-logs.ndjson"
import_response="/tmp/kibana-import-response.json"

echo "Waiting for Kibana"
until curl -fsS "${kibana_url}/api/status" >/dev/null; do
  sleep 5
done

echo "Creating Kibana data view"
curl -fsS -X POST "${kibana_url}/api/data_views/data_view" \
  -H "Content-Type: application/json" \
  -H "kbn-xsrf: true" \
  -d '{
    "override": true,
    "data_view": {
      "id": "subscription-service-logs",
      "title": "filebeat-*",
      "name": "Subscription Service Logs",
      "timeFieldName": "@timestamp"
    }
  }' >/dev/null

echo "Importing Kibana log dashboard"
curl -fsS -X POST "${kibana_url}/api/saved_objects/_import?overwrite=true" \
  -H "kbn-xsrf: true" \
  --form "file=@${saved_objects_file}" > "${import_response}"

cat "${import_response}"
if ! grep -q '"success":true' "${import_response}"; then
  echo "Kibana saved object import failed" >&2
  exit 1
fi

echo "Kibana observability resources are ready"
