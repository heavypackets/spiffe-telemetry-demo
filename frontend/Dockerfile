FROM rnburn/envoy-jaeger:0.11

RUN apt-get update && \
    apt-get install -qqy \
    ca-certificates gettext python

# Install Envoy config
COPY frontend/envoy.json /etc/envoy.json
COPY frontend/start_envoy.sh /usr/local/bin/start_envoy.sh
COPY ls_access_token /etc/ls_access_token
RUN chmod +x /usr/local/bin/start_envoy.sh

# Install Jaeger config
COPY jaeger.yaml /etc/jaeger.yaml
RUN ln -s /usr/local/lib/libjaegertracingd.so /usr/local/lib/libjaegertracing.so 

# Install SPIRE agent
ARG SPIRE_VERSION=0.6.0
ARG SPIRE_RELEASE=https://github.com/spiffe/spire/releases/download/${SPIRE_VERSION}/spire-${SPIRE_VERSION}-linux-x86_64-glibc.tar.gz
ARG SPIRE_DIR=/opt/spire

RUN curl --silent --location $SPIRE_RELEASE | tar -xzf -
RUN mv spire-${SPIRE_VERSION} ${SPIRE_DIR}

WORKDIR ${SPIRE_DIR}
COPY spire/conf/agent.conf conf/agent/agent.conf

# Install sidecar
COPY spire/spiffe-helper /usr/local/bin/spiffe-helper
COPY spire/helper.conf ${SPIRE_DIR}
COPY envoy/hot-restarter.py /usr/local/bin/
RUN mkdir -p /certs && chmod 777 /certs

EXPOSE 80
EXPOSE 8001

ENTRYPOINT ["/bin/bash"]