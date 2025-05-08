# medias-ms-go
Performant microservice to manage files uploads via a neat API

## Local setup

- update ``.env`` with any ``MINIO_USER`` and ``MINIO_PASS`` (password has to be eight characters min.)
- ``docker-compose up -d``
- go to http://localhost:9001/
- login using ``MINIO_USER`` and ``MINIO_PASS``
- go to ``Access Keys`` in the lateral menu, click ``Create access key``. It will generate both access and secret keys, update ``MINIO_ACCESS_KEY`` and ``MINIO_SECRET_KEY`` in your ``.env``
- go to ``Buckets`` in the lateral menu, create as many buckets as you want. You need at least one called ``staging`` that will be used for validation before moving the files to their final destination bucket