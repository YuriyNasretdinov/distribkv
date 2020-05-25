#!/bin/bash
set -e

trap 'killall distribkv' SIGINT

cd $(dirname $0)

killall distribkv || true
sleep 0.1

go install -v

distribkv -db-location=moscow.db -http-addr=127.0.0.1:8080 -config-file=sharding.toml -shard=Moscow &
distribkv -db-location=minsk.db -http-addr=127.0.0.1:8081 -config-file=sharding.toml -shard=Minsk &
distribkv -db-location=kiev.db -http-addr=127.0.0.1:8082 -config-file=sharding.toml -shard=Kiev &
distribkv -db-location=tashkent.db -http-addr=127.0.0.1:8083 -config-file=sharding.toml -shard=Tashkent &

wait
