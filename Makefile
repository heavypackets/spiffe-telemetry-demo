all: backend frontend spire

backend: backend/donutbin
	docker build -t donutz/donutbin --file backend/Dockerfile .

backend/donutbin:
	cd backend && ./build.sh

frontend:
	docker-compose build front-envoy

spire:
	docker-compose build spire-server

env: clean backend frontend spire
	docker-compose down
	./spire/register_nodes.sh

clean:
	rm -f backend/donutbin
	docker-compose rm

.PHONY: backend frontend spire env
