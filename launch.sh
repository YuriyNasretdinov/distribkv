#!/bin/bash
set -e

trap 'killall distribkv' SIGINT

cd $(dirname $0)

killall distribkv || true
sleep 0.1

go install -v

distribkv -db-location=moscow.db -http-addr=127.0.0.2:8080 -config-file=sharding.toml -shard=Moscow &
distribkv -db-location=moscow-r.db -http-addr=127.0.0.22:8080 -config-file=sharding.toml -shard=Moscow -replica &

distribkv -db-location=minsk.db -http-addr=127.0.0.3:8080 -config-file=sharding.toml -shard=Minsk &
distribkv -db-location=minsk-r.db -http-addr=127.0.0.33:8080 -config-file=sharding.toml -shard=Minsk -replica &

distribkv -db-location=kiev.db -http-addr=127.0.0.4:8080 -config-file=sharding.toml -shard=Kiev &
distribkv -db-location=kiev-r.db -http-addr=127.0.0.44:8080 -config-file=sharding.toml -shard=Kiev -replica &

distribkv -db-location=tashkent.db -http-addr=127.0.0.5:8080 -config-file=sharding.toml -shard=Tashkent &
distribkv -db-location=tashkent-r.db -http-addr=127.0.0.55:8080 -config-file=sharding.toml -shard=Tashkent -replica &

wait
