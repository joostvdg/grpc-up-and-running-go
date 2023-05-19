# OpenTelemetry Demo

Take from: https://github.com/open-telemetry/opentelemetry-demo/tree/main
The docs are here: https://opentelemetry.io/docs/demo/docker-deployment/

## Run

```shell
docker-compose up --no-build
```

## Use Env Variables for OTEL

```shell
export OTEL_SERVICE_NAME=ProductInfoService
export OTEL_RESOURCE_ATTRIBUTES="service.namespace=opentelemetry-demo,service.name=ProductInfoService,service.instance.id=ProductInfoService-1"
```

And there are these as well:

* `OTEL_EXPORTER_OTLP_ENDPOINT`
* `OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE`
* `OTEL_RESOURCE_ATTRIBUTES`

See: https://opentelemetry.io/docs/specs/otel/sdk-environment-variables/

### In Go GRPC Code

```go
srv := grpc.NewServer(
    grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
    grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
)
```

## Open in Browser

* Grafana: http://localhost:8080/grafana/
* Jaeger: http://localhost:8080/jaeger/