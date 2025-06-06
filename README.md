# medias-ms-go

Performant microservice to manage files uploads via a neat API for serving to web applications.

All files uploaded are asynchronously optimised, while still being immediately available in their original state. Images are transformed into ``.webp``, as well as declined in resized variants depending on the sizes given in the ``IMAGES_SIZES`` environment variable.

Currently accepts PNG | JPG | WEBP | PDF | MD.

# ***STILL WIP, NOT READY FOR PROD USAGE***

## Requirements
- Go CLI
- Docker


- *if on Windows,* [GCC/MinGW](https://jmeubank.github.io/tdm-gcc/download/)


- *(optional)* a Redis server, used for cache and file optimisations
- *(optional)* Make, for nice and easy commands

## Local setup

- copy ``.env.dist`` to a new ``.env``, update any value you feel like changing
- run ``docker-compose up -d``
- run migrations with ``go run ./cmd/migrate/``
- run the server with ``go run ./cmd/api/``

## Notes

- Missing buckets from the list given in the ``BUCKETS`` env variable will be created automatically on application startup
- The ``staging`` bucket is mandatory and will be created even when missing from the env variable. It is used to temporarily host files waiting for validation
- You have access to a UI for browsing MinIO buckets and objects at [localhost:9001](http://localhost:9001), using the login info from ``MINIO_USER`` / ``MINIO_PASS`` in the env variables

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
- Set `REDIS_ADDR` in your environment (typically something like ``localhost:6379``)
