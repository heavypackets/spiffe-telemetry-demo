FROM rnburn/envoy-jaeger:0.15

RUN apt-get update && \
    apt-get install -qqy \
    ca-certificates gettext dnsutils python curl

# Install app
COPY backend/donutbin /usr/local/bin/donutbin
COPY backend/go/src/app /etc/app
COPY backend/start_back.sh /usr/local/bin/start_back.sh
RUN chmod +x /usr/local/bin/start_back.sh

# Install SPIRE agent
ARG SPIRE_VERSION=0.6.0
ARG SPIRE_RELEASE=https://github.com/spiffe/spire/releases/download/${SPIRE_VERSION}/spire-${SPIRE_VERSION}-linux-x86_64-glibc.tar.gz
ARG SPIRE_DIR=/opt/spire

RUN curl --silent --location $SPIRE_RELEASE | tar -xzf -
RUN mv spire-${SPIRE_VERSION} ${SPIRE_DIR}

WORKDIR ${SPIRE_DIR}
COPY spire/conf/agent.conf conf/agent/agent.conf

# Install envoy config
COPY backend/envoy.json /etc/envoy.json
COPY backend/start_envoy.sh /usr/local/bin/start_envoy.sh
RUN chmod +x /usr/local/bin/start_envoy.sh

# Install Jaeger config
COPY jaeger.yaml /etc/jaeger.yaml
RUN ln -s /usr/local/lib/libjaegertracingd.so /usr/local/lib/libjaegertracing.so

# Install sidecar
COPY spire/spiffe-helper /usr/local/bin/spiffe-helper
COPY spire/helper.conf ${SPIRE_DIR}
COPY envoy/hot-restarter.py /usr/local/bin/
RUN mkdir -p /certs && chmod 777 /certs

EXPOSE 80
EXPOSE 8001
EXPOSE 8080

ENTRYPOINT start_back.sh
