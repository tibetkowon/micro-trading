#!/bin/bash
# DuckDNS IP 업데이트 스크립트
# 필수 환경변수: DUCKDNS_TOKEN, DUCKDNS_DOMAIN

LOGFILE="/var/log/duckdns.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

if [ -z "$DUCKDNS_TOKEN" ] || [ -z "$DUCKDNS_DOMAIN" ]; then
    echo "[$TIMESTAMP] ERROR: DUCKDNS_TOKEN or DUCKDNS_DOMAIN not set" >> "$LOGFILE"
    exit 1
fi

RESPONSE=$(curl -s "https://www.duckdns.org/update?domains=${DUCKDNS_DOMAIN}&token=${DUCKDNS_TOKEN}&ip=")

if [ "$RESPONSE" = "OK" ]; then
    CURRENT_IP=$(curl -s https://api4.ipify.org)
    echo "[$TIMESTAMP] OK - IP updated to $CURRENT_IP" >> "$LOGFILE"
else
    echo "[$TIMESTAMP] FAIL - response: $RESPONSE" >> "$LOGFILE"
    exit 1
fi
