FROM alpine as certs
RUN apk update && apk add ca-certificates

FROM golang:1.19-alpine AS builder

WORKDIR /builder

COPY . .

# build all
RUN CGO_ENABLED=0 go build -o workflow --mod=vendor -ldflags='-s -w'  -installsuffix cgo main.go
RUN CGO_ENABLED=0 go build -o node-email --mod=vendor -ldflags='-s -w'  -installsuffix cgo pkg/node/nodes/email/main.go
RUN CGO_ENABLED=0 go build -o node-examine --mod=vendor -ldflags='-s -w'  -installsuffix cgo pkg/node/nodes/examine/main.go
RUN CGO_ENABLED=0 go build -o node-processBranch --mod=vendor -ldflags='-s -w'  -installsuffix cgo pkg/node/nodes/process_branch/main.go
RUN CGO_ENABLED=0 go build -o node-formCreate --mod=vendor -ldflags='-s -w'  -installsuffix cgo pkg/node/nodes/quanxiang_form/create/main.go
RUN CGO_ENABLED=0 go build -o node-formUpdate --mod=vendor -ldflags='-s -w'  -installsuffix cgo pkg/node/nodes/quanxiang_form/update/main.go
RUN CGO_ENABLED=0 go build -o node-webHook --mod=vendor -ldflags='-s -w'  -installsuffix cgo pkg/node/nodes/webhook/main.go

RUN CGO_ENABLED=0 go build -o quanxiangAdapter -ldflags='-s -w'  -installsuffix cgo pkg/mid/main.go

FROM scratch
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

WORKDIR /apps
COPY --from=builder ./builder/workflow .
COPY --from=builder ./builder/node-email .
COPY --from=builder ./builder/node-examine .
COPY --from=builder ./builder/node-processBranch .
COPY --from=builder ./builder/node-formCreate .
COPY --from=builder ./builder/node-formUpdate .
COPY --from=builder ./builder/node-webHook .
COPY --from=builder ./builder/quanxiangAdapter .


