from apache/impala:81d5377c2-impala_quickstart_client

USER root
# install azurecli
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# install azure-storage-blob python lib
RUN pip install azure-storage-blob

COPY b3 /usr/local/bin
COPY impala /usr/local/bin
COPY gen-schema.sh /usr/local/bin
RUN mkdir -p /opt/impala/data && chmod a+rwx /opt/impala/data

USER impala
RUN cd data
