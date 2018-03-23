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
