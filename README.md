# calc-image-size

Calculates the size of an image on a Docker registry.
It takes into account image indexes (images for all platforms), image manifests and all the layers.
Currently it doesn't take into account the size of image configs. But its likely that thier size if negligible compared to the actual images.

## Install:
```
go install github.com/dvob/calc-image-size@latest
```

## Usage:
```
calc-image-size IMAGE1 IMAGE2 ... IMAGEN
```

If images do not have a specifc tag, all tags are used.

## Examples:
Calculate only specific `latest` tag for `busybox`  but all tags for `dvob/http-server`
```
calc-image-size busybox:latest dvob/http-server
```
