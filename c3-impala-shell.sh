#!/bin/bash

docker run --network=quickstart-network -it \
     ${IMPALA_QUICKSTART_IMAGE_PREFIX}impala_quickstart_client impala-shell
