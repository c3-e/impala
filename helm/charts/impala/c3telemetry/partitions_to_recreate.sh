#!/bin/bash

export days=${1-3}
export base=/opt/impala/c3telemetry

cat $base/clusters.csv | xargs -I {} $base/days_to_refresh.sh {} $days | \
  xargs -l bash -c 'echo ALTER TABLE c3telemetry.raw DROP IF EXISTS PARTITION \(cluster=~$0~, yyyy=~$1~, mm=~$2~, dd=~$3~\)\; ALTER TABLE c3telemetry.raw ADD PARTITION \(cluster=~$0~, yyyy=~$1~, mm=~$2~, dd=~$3~\) LOCATION ~abfs://c3telemetry@stageazpaasstore01.dfs.core.windows.net/$0/yyyy=$1/mm=$2/dd=$3~\;' | \
  sed "s/~/'/g"
