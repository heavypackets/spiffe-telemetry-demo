#!/bin/bash

docker-compose up -d
sleep 8

docker-compose exec -d donutsalon-1 sh -c "cd /opt/spire && ./spire-agent run -joinToken $(docker-compose exec spire-server sh -c 'cd /opt/spire && ./spire-server token generate -spiffeID spiffe://example.org/donutsalon-1' | sed 's/Token: //' | sed "s/$(printf '\r')//")"
docker-compose exec -d donutsalon-2 sh -c "cd /opt/spire && ./spire-agent run -joinToken $(docker-compose exec spire-server sh -c 'cd /opt/spire && ./spire-server token generate -spiffeID spiffe://example.org/donutsalon-2' | sed 's/Token: //' | sed "s/$(printf '\r')//")"
docker-compose exec -d donutsalon-3 sh -c "cd /opt/spire && ./spire-agent run -joinToken $(docker-compose exec spire-server sh -c 'cd /opt/spire && ./spire-server token generate -spiffeID spiffe://example.org/donutsalon-3' | sed 's/Token: //' | sed "s/$(printf '\r')//")"
docker-compose exec -d front-envoy sh -c "cd /opt/spire && ./spire-agent run -joinToken $(docker-compose exec spire-server sh -c 'cd /opt/spire && ./spire-server token generate -spiffeID spiffe://example.org/front-envoy' | sed 's/Token: //' | sed "s/$(printf '\r')//")"
docker-compose exec spire-server sh -c "cd /opt/spire && ./spire-server entry create -parentID spiffe://example.org/donutsalon-1 -spiffeID spiffe://example.org/backend1 -selector unix:uid:0 -ttl 2400"
docker-compose exec spire-server sh -c "cd /opt/spire && ./spire-server entry create -parentID spiffe://example.org/donutsalon-2 -spiffeID spiffe://example.org/backend2 -selector unix:uid:0 -ttl 2400"
docker-compose exec spire-server sh -c "cd /opt/spire && ./spire-server entry create -parentID spiffe://example.org/donutsalon-3 -spiffeID spiffe://example.org/backend3 -selector unix:uid:0 -ttl 2400"
docker-compose exec spire-server sh -c "cd /opt/spire && ./spire-server entry create -parentID spiffe://example.org/front-envoy -spiffeID spiffe://example.org/frontend -selector unix:uid:0 -ttl 2400"

docker-compose exec -d donutsalon-1 sh -c "cd /opt/spire && spiffe-helper"
docker-compose exec -d donutsalon-2 sh -c "cd /opt/spire && spiffe-helper"
docker-compose exec -d donutsalon-3 sh -c "cd /opt/spire && spiffe-helper"
docker-compose exec -d front-envoy sh -c "cd /opt/spire && spiffe-helper"
