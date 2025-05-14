FROM golang:1.24 AS build

COPY . /src
WORKDIR /src
RUN CGO_ENABLED=0 GOOS=linux go build -o cloud-balancer

FROM scratch

COPY --from=build /src/cloud-balancer .
COPY --from=build /src/.env .
COPY --from=build /src/servers.yaml .

EXPOSE 8080
CMD ["/cloud-balancer"]
