compose="docker compose -f docker-compose-bulk.yml"
$compose down > /dev/null 2>&1
docker volume rm bench_dbvolume > /dev/null 2>&1

$compose up --build | grep Loaded | tee bench.log

duration=`grep -m1 "10000\sdocuments" bench.log | awk '{print $7}' | cut -d',' -f1`

echo Load time: $duration
