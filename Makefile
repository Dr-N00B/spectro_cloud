.PHONY=deploy build clean build_app build_image

REGISTRY_URL=arun0110
REPO=spectro_cloud
TAG=v1.0
REPO_WITH_TAG=$(REPO):$(TAG)
POD_FILE=./k8/deploy.yaml
TEST_DIR=./k8/tests

all: deploy

push: build
	docker push $(REGISTRY_URL)/$(REPO_WITH_TAG)

build: build_app build_image

build_app:
	go mod tidy -v
	go build -o ./bin/assignment main.go

build_image:
	docker build -f Dockerfile -t $(REGISTRY_URL)/$(REPO_WITH_TAG) .

deploy:
	kubectl apply -f $(POD_FILE)

delete:
	kubectl delete -f $(POD_FILE)

clean: delete_tests
	rm -rf ./bin
	docker rmi $(REGISTRY_URL)/$(REPO_WITH_TAG)

delete_tests: $(TEST_DIR)/*
	@for file in $^ ; do kubectl delete -f $${file} ; done
 
tests: $(TEST_DIR)/*
	@for file in $^ ; do kubectl apply -f $${file} ; done