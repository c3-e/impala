from apache/impala:81d5377c2-impala_quickstart_client

USER root
# install azurecli
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# install azure-storage-blob python lib
RUN pip install azure-storage-blob

COPY b3 /usr/local/bin
COPY impala /usr/local/bin

USER impala
