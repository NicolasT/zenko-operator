FROM golang:1.10 as builder
ARG HELM_CHART
ARG API_VERSION
ARG KIND
WORKDIR /go/src/github.com/NicolasT/zenko-operator
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/operator cmd/zenko-operator/main.go
RUN chmod +x bin/operator

FROM alpine:3.6
ARG HELM_CHART
ARG API_VERSION
ARG KIND
ENV API_VERSION $API_VERSION
ENV KIND $KIND
WORKDIR /
COPY --from=builder /go/src/github.com/NicolasT/zenko-operator/bin/operator /operator
COPY charts /charts
ENV HELM_CHART /charts/zenko

CMD ["/operator"]
