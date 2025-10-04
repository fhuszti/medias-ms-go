# medias-ms-go

Performant microservice to manage files uploads via a neat API for serving to web applications.

All files uploaded are asynchronously optimised, while still being immediately available in their original state. Images are transformed into ``.webp``, as well as declined in resized variants depending on the sizes given in the ``IMAGES_SIZES`` environment variable.

Currently accepts PNG | JPG | WEBP | PDF | MD.

### ***This microservice has been fully tested and is working blazing fast for local dev, but was never tested in prod nor at scale.***

## Requirements
- Go CLI
- Docker


- *if on Windows,* [GCC/MinGW](https://jmeubank.github.io/tdm-gcc/download/)


- *(optional)* a Redis server, used for cache and file optimisations
- *(optional)* Make, for nice and easy commands

## Local setup

- copy ``.env.dist`` to a new ``.env``, update any value you feel like changing
- run ``docker-compose up -d``
- run migrations with ``make migrate``
- run the server with ``make start``

## Notes

- Missing buckets from the list given in the ``BUCKETS`` env variable will be created automatically on application startup
- The ``staging`` bucket is mandatory and will be created even when missing from the env variable. It is used to temporarily host files waiting for validation
- You have access to a UI for browsing MinIO buckets and objects at [localhost:9001](http://localhost:9001), using the login info from ``MINIO_USER`` / ``MINIO_PASS`` in the env variables

## Basic API usage

1. **Generate an upload link** – ``POST /medias/generate_upload_link``
   - Body: ``{"name": "original-file.ext"}``
   - Returns ``201`` with ``{"id":"<uuid>","url":"<upload_url>"}``.
2. **Upload the file** using the ``url`` from step 1 with a ``PUT`` request.
   - The file lands in the ``staging`` bucket.
3. **Finalise the upload** – ``POST /medias/finalise_upload/{id}``
   - ``id`` is the media ID returned in step 1.
   - Body: ``{"dest_bucket": "<bucket>"}`` where ``dest_bucket`` must match one of the buckets from ``BUCKETS``.
   - Moves the file from ``staging`` to ``dest_bucket`` and stores metadata.
   - Returns ``204`` with no content.
4. **Retrieve the media** – ``GET /medias/{id}``
   - Returns ``200`` with ``{"valid_until":"<time>","optimised":<bool>,"url":"<download_url>","metadata":{...},"variants":[]}``.
   - ``metadata`` always contains ``size_bytes`` and ``mime_type``.
     - **Images**: also include ``width`` and ``height``.
     - **PDFs**: ``metadata`` has ``page_count``.
     - **Markdown**: ``metadata`` has ``word_count``, ``heading_count``, ``link_count``.
   - ``variants`` lists resized ``.webp`` versions for images. Each variant has ``url``, ``width``, ``height``, ``size_bytes``. Other file types return an empty list.

### Async optimisations

 After step 3 the service enqueues optimisation tasks handled by the worker (requires Redis):

- The original file is compressed (images become lossy ``.webp``; PDFs are stripped, etc.).
- If the resulting file is an image, resized variants are created for the sizes in ``IMAGES_SIZES``.
- When a size in ``IMAGES_SIZES`` exceeds the original width, that variant is generated as an untouched copy so the image is never stretched.
- These operations run in the background so the original file remains available immediately after finalisation.
  While processing, ``optimised`` in ``GET /medias/{id}`` stays ``false`` and ``variants`` is empty. Once compression and resizing finish, ``optimised`` becomes ``true`` and image variants are listed.

## Manual commands

- run the server with ``make start`` (``go run ./cmd/api/``)
- start the worker with ``make worker`` (``go run ./cmd/worker/``) *(requires Redis)*
- run database migrations with ``make migrate`` (``go run ./cmd/migrate/``)
- run the backlog optimiser with ``make optimise-backlog`` (``go run ./cmd/optimise-backlog/``) *(requires Redis)*

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

## JWT authentication *(optional)*

If `JWT_PUBLIC_KEY_PATH` is set in the environment, the API requires all requests to
include a valid JWT token as a Bearer token in the `Authorization` header. The
token signature is verified using this RSA public key. When `JWT_PUBLIC_KEY_PATH` is
empty, authentication is skipped and all requests are allowed through.
