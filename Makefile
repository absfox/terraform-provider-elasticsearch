
NAME = terraform-provider-elasticsearch

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
PLUGIN_DIR = ~/.terraform.d/plugins/$(GOOS)_$(GOARCH)

INT_TEST_ENV = ELASTICSEARCH_URL=http://localhost:9200 TF_ACC=TRUE

.PHONY: clean
clean:
	rm -f $(NAME)

.PHONY: test
test:
	go test -cover . -count=1

.PHONY: integration-test
integration-test:
	$(INT_TEST_ENV) go test -v -cover . -run TestAccElasticsearchIndex -count=1

$(NAME):
	go build -o $(NAME) .

install: $(NAME)
	stat $(PLUGIN_DIR) 2>&1 > /dev/null || mkdir -p $(PLUGIN_DIR)
	cp $(NAME) $(PLUGIN_DIR)

all: $(NAME)
