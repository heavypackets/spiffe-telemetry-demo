#!/usr/bin/env bash
echo "donutsalon-1:" && ./spire-server token generate -spiffeID spiffe://example.org/donutsalon-1
echo "front-envoy:" && ./spire-server token generate -spiffeID spiffe://example.org/front-envoy

./spire-server entry create -parentID spiffe://example.org/donutsalon-1 -spiffeID spiffe://example.org/backend1 -selector unix:uid:0 -ttl 2400
./spire-server entry create -parentID spiffe://example.org/front-envoy -spiffeID spiffe://example.org/frontend -selector unix:uid:0 -ttl 2400
