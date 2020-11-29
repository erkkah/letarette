#!/bin/sh

start=`grep "\s0\sdocs" bench.log | tail -1 | awk '{print $4, $5}'`
end=`grep -m1 "\s10000\sdocs" bench.log | awk '{print $4, $5}'`

start_sec=`date -D %Y/%m/%d%t%T -d "$start" +%s`
end_sec=`date -D %Y/%m/%d%t%T -d "$end" +%s`

echo Load time: $((end_sec - start_sec)) seconds
