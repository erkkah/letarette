#!/bin/sh

docker compose down > /dev/null 2>&1
docker volume rm bench_dbvolume > /dev/null 2>&1

OS=`uname -s`
if [ $OS == Darwin ]; then
    OPT=--line-buffered
fi

docker compose -f docker-compose.yml up --build -d
docker compose logs -f | grep $OPT monitor | tee bench.log | \
    (grep $OPT -m1 '10000 docs' && docker compose down)

docker run --rm -v ${PWD}:/bench -w /bench alpine ./load-report.sh
