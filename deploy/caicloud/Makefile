TAGS = canary
PREFIX = index.caicloud.io/caicloud

compile:
	GOOS=linux GOARCH=amd64  godep go build -v ../../...
	GOOS=linux GOARCH=amd64  godep go build -v -o heapster k8s.io/heapster/metrics
	GOOS=linux GOARCH=amd64  godep go build -v -o eventer k8s.io/heapster/events

build: 
	docker build -t $(PREFIX)/heapster:$(TAGS) -f Dockerfile .

influxdb:
	docker build -t $(PREFIX)/heapster_influxdb:$(TAGS) -f Dockerfile-influxdb .

release: compile build
	docker push $(PREFIX)/heapster:$(TAGS)

