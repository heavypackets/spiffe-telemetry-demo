all: backend frontend spire

backend: backend/donutbin
	sudo docker build -t donutz/donutbin --file backend/Dockerfile .

backend/donutbin:
	cd backend && ./build.sh

frontend:
	sudo docker-compose build front-envoy

spire:
	sudo docker-compose build spire-server

env: clean backend frontend spire
	sudo docker-compose down
	sudo ./spire/register_nodes.sh

clean:
	rm -f backend/donutbin
	sudo docker-compose rm

.PHONY: backend frontend spire env
