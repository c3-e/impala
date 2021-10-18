from apache/impala:81d5377c2-impala_quickstart_client

USER root
# install azurecli
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# install azure-storage-blob python lib
RUN pip install azure-storage-blob

# install golang
# RUN apt -y update && apt -y upgrade && apt -y install golang-go
RUN VERSION="1.16.4" && ARCH="amd64" && \
    curl -O -L "https://golang.org/dl/go${VERSION}.linux-${ARCH}.tar.gz" && \
    tar -xf "go${VERSION}.linux-${ARCH}.tar.gz" && \
    mv go /usr/local && \
    rm -f "go${VERSION}.linux-${ARCH}.tar.gz"

# install b3
COPY b3 /usr/local/bin
COPY impala /usr/local/bin
COPY gen-schema.sh /usr/local/bin
RUN mkdir -p /opt/impala/data && chmod a+rwx /opt/impala/data

RUN mkdir /home/impala && chown impala:impala /home/impala

# install golang services
RUN mkdir -p /opt/impala/bin && chmod a+rwx /opt/impala/bin
COPY wrapper.sh /opt/impala/bin/wrapper.sh
COPY golog/ /opt/impala/bin/golog/
COPY goterm/ /opt/impala/bin/goterm/
RUN cd /opt/impala/bin/golog && /usr/local/go/bin/go build -o golog 
RUN cd /opt/impala/bin/goterm && /usr/local/go/bin/go build -o goterm && \
    cd /opt/impala/data

USER impala

RUN echo 'export GOPATH=$HOME/go' > /home/impala/.bashrc && \
    echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> /home/impala/.bashrc
