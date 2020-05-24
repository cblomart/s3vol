# s3vol

Docker Volume plugin using s3fs

## install

Should install by docker plugin install cblomart/s3vol:edge

> TODO: test and document

sample install:
```bash
> docker plugin install --alias s3vol cblomart/s3vol:edge-arm  S3VOL_ACCESSKEY=rp1mini0 S3VOL_SECRETKEY=83449e8a262cbab3513d7ff713de9a9bfb0bc106 S3VOL_ENDPOINT=http://localhost:9000/ S3VOL_DEFAULTS=allow_other,mp_umask=0022,use_cache=/tmp/s3fs/,gid=0,uid=0
```
