# jasmin

The new ZERM backend. :sparkle:

## Deployment/Configuration

| Environment Variable | Default Value      | Short Description                                                                                         |
|----------------------|--------------------|-----------------------------------------------------------------------------------------------------------|
| `JASMIN_ROOT_DIR`    | `/var/www/zerm.eu` | Base directory where jasmin files and data are stored.                                                    |
| `JASMIN_HTTP_ADDR`   | `:8099`            | Address (host:port) on which jasmin listens for plain HTTP requests.                                      |
| `JASMIN_HTTPS_ADDR`  | _none_             | Address (host:port) on which jasmin listens for HTTPS requests. Leave unset if HTTPS is not used.         |
| `JASMIN_CERT_FILE`   | _none_             | File path to the TLS/SSL certificate used for HTTPS. Required if `JASMIN_HTTPS_ADDR` is set.              |
| `JASMIN_KEY_FILE`    | _none_             | File path to the private key corresponding to `JASMIN_CERT_FILE`. Required if `JASMIN_HTTPS_ADDR` is set. |

Using an HTTPS and caching reverse-proxy is highly advised, but if you want to leave it out, you can use the last 3 variables.

### System package managers

`apk`, `dpkg` and `rpm` packages can be downloaded from
[the Releases tab](https://github.com/ZERMZeitung/jasmin/releases).
Alternatively, you can install the binary from the `tar.gz` archive.
Service management is up to the system administrator.

### Docker (OCI)

> [!WARNING]
> Docker deployment is currently being implemented as part of chrissx Media's
> Project SHACS. It is not production ready or well-tested, use with caution.

Guide TBD.
