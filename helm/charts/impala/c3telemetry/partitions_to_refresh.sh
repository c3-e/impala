#!/bin/bash

export days=${1-3}
export base=/opt/impala/c3telemetry

cat $base/clusters.csv | xargs -I {} $base/days_to_refresh.sh {} $days | \
  xargs -l bash -c 'echo REFRESH c3telemetry.raw PARTITION \(cluster=~$0~, yyyy=~$1~, mm=~$2~, dd=~$3~\)\;' | \
  sed "s/~/'/g"
