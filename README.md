# DonutSalon

## Setup

```
brew install direnv
direnv allow
```

## Deploy DonutSalon with Docker-Compose (LightStep edition)

```
# run build once, and after any change to app
./build.sh
docker-compose up
```

**Pro Tip:** To generate trace data, you can add fake requests by uncommenting these lines:
https://github.com/lightstep/donutsalon/blob/master/go/src/app/main.go#L81

## Prometheus Queries

donutshop_chocolate_donuts_stock and on (instance, job) donutshop_app_identity{spiffe_id="spiffe://example.org/backend2"}

donutshop_chocolate_donuts_stock * on (instance, job) group_left(spiffe_id) donutshop_app_identity

donutshop_total_ordered_donuts * on (instance, job) group_left(spiffe_id) donutshop_app_identity{spiffe_id="spiffe://example.org/backend1"}