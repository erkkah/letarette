docker-compose down > /dev/null 2>&1
docker volume rm bench_dbvolume > /dev/null 2>&1

docker-compose -f docker-compose.yml up -d
docker-compose logs -f | grep monitor | tee bench.log | \
    (grep -m1 '10000 docs' && docker-compose down)

start=`grep -m1 "\s0\sdocs" bench.log | awk '{print $4, $5}'`
end=`grep -m1 "\s10000\sdocs" bench.log | awk '{print $4, $5}'`

start_sec=`date -d "$start" +%s`
end_sec=`date -d "$end" +%s`

echo Load time: $((end_sec - start_sec))
