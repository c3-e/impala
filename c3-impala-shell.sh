#!/bin/bash

docker run --network=quickstart-network -it \
     ${IMPALA_QUICKSTART_IMAGE_PREFIX}impala_quickstart_client impala-shell -l --auth_creds_ok_in_clear -u impala --ldap_password_cmd="echo -n c3impala"

## bash first then run impala-shell
# docker run --network=quickstart-network -it      ${IMPALA_QUICKSTART_IMAGE_PREFIX}impala_quickstart_client bash
# impala-shell -i impalad-1:21000 -l --auth_creds_ok_in_clear -u impala --ldap_password_cmd="echo -n c3impala"

## no authentication
# docker run --network=quickstart-network -it \
#     ${IMPALA_QUICKSTART_IMAGE_PREFIX}impala_quickstart_client impala-shell
