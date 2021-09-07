#!/bin/bash

export TABLE_NAME=${TABLE_NAME-sean_test.raw}
export BASE_LOCATION=${BASE_LOCATION-abfs://sean-test@stageazpaasstore01.dfs.core.windows.net}

cat << EOF
create external table $TABLE_NAME(
  InstanceId string,
  InstanceType string,
  CpuCount string,
  GpuCount string,
  ts string,
  Uptime string,
  Metric string,
  Resource string,
  C3Id string,
  Value string,
  Provider string,
  Env string,
  Customer string,
  Region string
) partitioned by (cluster string, dt string)
row format delimited fields terminated by ','
stored as textfile;
EOF

function partition() {
  if [ -n "$2" ]; then # folder depth of 2
    cat << EOF
ALTER TABLE $TABLE_NAME ADD PARTITION (cluster='$1', dt='$2')
LOCATION '$BASE_LOCATION/$1/$2';
EOF
  fi
}
export -f partition

sed 's/.*://' <&0 | tr '/' ' ' | xargs -l bash -c 'partition $0 $1'
