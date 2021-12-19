## Sidecache


Sidecar cache for kubernetes applications. It acts as a proxy sidecar between application and client, routes incoming requests to cache storage or application according to Istio VirtualService routing rules.


## Diagram

![arch](https://wiki.trendyol.com/download/attachments/51283813/image_2021-03-09_081136.png?version=1&modificationDate=1615266696671&api=v2)

## How to video

[video](https://drive.google.com/drive/u/0/folders/1YTLk3Al0KXG40tnLnEasz1Xur1pJlA5c)



## Integration

We use the Kubernetes admission webhook controller to enable the sidecache feature that intercepts requests to the Kubernetes API server prior to persistence of the object.

### 1- Add Sidecache Annotation to Deployment

First step is adding annotation to deployment under **spec-> template-> metadata-> annotation**

Annotation: **sidecache.trendyol.com/inject: "true"**


Important Note!
- Add **sidecache.trendyol.com/port** annotation with a port value  if you want to run sidecache on another port (default is 9191).
- MAIN_CONTAINER_PORT environment variable: Sidecache takes first container's port or 8080 as default.
- CACHE_KEY_PREFIX environment variable: Default is deployment name.

### 2- Add Port Definition to Service

The sidecache port definition must be added to your application service yaml.So that we can make a smart route for where the request goes.
