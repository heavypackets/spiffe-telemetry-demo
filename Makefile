all: backend frontend spire

backend: backend/donutbin
	docker-compose build donutsalon-1

backend/donutbin:
	cd backend && ./build.sh

frontend:
	docker-compose build front-envoy

spire:
	docker-compose build spire-server

clean:
	rm -f backend/donutbin
	docker-compose rm

.PHONY: backend frontend spire
