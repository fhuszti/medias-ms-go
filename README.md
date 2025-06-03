# medias-ms-go

Performant microservice to manage files uploads via a neat API for serving to web applications.

All files uploaded are asynchronously optimised, while still being immediately available in their original state. Images are transformed into ``.webp``, as well as declined in resized variants depending on the sizes given in the ``IMAGES_SIZES`` environment variable.

Currently accepts PNG | JPG | WEBP | PDF | MD.

# ***STILL WIP, NOT READY FOR PROD USAGE***

## Requirements
- Go CLI
- Docker
- if on Windows, [GCC/MinGW](https://jmeubank.github.io/tdm-gcc/download/)


- *(optional)* a Redis server, used for cache and file optimisations
- *(optional)* Make, for nice and easy commands

## Local setup

- copy ``.env.dist`` to a new ``.env``


- update ``.env`` with any ``MINIO_USER`` and ``MINIO_PASS`` (password has to be eight characters min.)
- run ``docker-compose up -d``
- go to http://localhost:9001/
- login using ``MINIO_USER`` and ``MINIO_PASS``
- go to ``Access Keys`` in the lateral menu, click ``Create access key``. It will generate both access and secret keys, update ``MINIO_ACCESS_KEY`` and ``MINIO_SECRET_KEY`` in your ``.env``
- buckets that do not yet exist will be created automatically on server startup from the comma-separated list given in the ``.env`` in ``BUCKETS``


- run migrations with ``go run ./cmd/migrate/``
- run the server with either:
  - ``air`` (dev mode, with hot reload) 
  - or with ``go run ./cmd/api/``

## Tests

You can run the full tests suite with a simple ``make test``.
- ``make unit-tests`` to only run unit tests
- ``make functional-tests`` to run integrations and e2e tests

## Redis *(optional)*

Redis is used to enable:
- Background image optimization (resize, compress)
- Optional caching for faster media retrieval

If Redis is not configured:
- Media uploads will still work fully
- Optimisation will be skipped
- Cache layers will be bypassed (data always comes from DB)

**To enable Redis:**
- Set `REDIS_ADDR` in your environment