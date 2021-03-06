# sidecache

Sidecar cache for kubernetes applications. It acts as a proxy sidecar between application and client, routes incoming
requests to cache storage or application according to Istio VirtualService routing rules.


Medium
article: https://medium.com/trendyol-tech/trendyol-platform-team-caching-service-to-service-communications-on-kubernetes-istio-82327589b935

[![License: MIT](https://img.shields.io/badge/License-MIT-ligthgreen.svg)](https://opensource.org/licenses/MIT)

## Table Of Contents

- [Istio Configuration](#istio-configuration-for-routing-http-requests-to-sidecar-container)
- [Environment Variables](#environment-variables)

## Istio Configuration for Routing Http Requests to Sidecar Container

Below VirtualService is responsible for routing all get requests to port 9191 on your pod, other http requests goes to
port 8080.

```
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: foo
spec:
  gateways:
  - foo-gateway
  hosts:
  - foo
  http:
  - match:
    - method:
        exact: GET
    route:
    - destination:
        host: foo
        port:
          number: 9191
  - route:
    - destination:
        host: foo
        port:
          number: 8080
```

## Environment Variables

Environment variables for sidecar container.

- **MAIN_CONTAINER_PORT**: The port of main application to proxy.
- **COUCHBASE_HOST**: Couchbase host addr.
- **COUCHBASE_USERNAME**: Couchbase username.
- **COUCHBASE_PASSWORD**: Couchbase password.
- **BUCKET_NAME**: Couchbase cache bucket name.
- **CACHE_KEY_PREFIX**: Cache key prefix to prevent url conflicts between different applications.
- **SIDE_CACHE_PORT**: Sidecar container port to listen.

## Purging a cache

Sidecache provides a purge endpoint for removing cache.

### Example Request

#### [POST] /purge
Body
```json
{
  "url": "/users?age=12&gender=man"
}
```

### FAQ 

https://gitlab.trendyol.com/platform/base/apps/platform-faq/-/blob/master/docs/sidecache.md
