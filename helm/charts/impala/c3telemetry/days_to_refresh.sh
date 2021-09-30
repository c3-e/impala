#!/bin/bash

export days=${2-3}

from=$(date -I -d "$d - $2 day")
to=$(date -I -d "$d + 1 day")
d=$from
while [ "$d" != $to ]; do
  echo $1 $(date +%Y -d $d) $(date +%m -d $d) $(date +%d -d $d)
  d=$(date -I -d "$d + 1 day")
done
