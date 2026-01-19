#!/bin/bash

SERVER="http://192.168.10.167"

if [ -z "$1" ]; then
  echo "Usage: $0 <path-to-excel-file>"
  exit 1
fi

if [ ! -f "$1" ]; then
  echo "Error: File not found: $1"
  exit 1
fi

echo "Uploading $1 to $SERVER..."

curl -X POST "$SERVER/api/upload" \
  -F "file=@$1" \
  -F "type=cost-ledger"

echo ""
