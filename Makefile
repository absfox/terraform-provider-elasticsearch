
NAME = terraform-provider-elasticsearch

.PHONY: clean
clean:
	rm -f $(NAME)

$(NAME):
	go build -o $(NAME) .

.PHONY: test
test:
	go test -cover . -count=1

.PHONY: intest
intest:
	ELASTICSEARCH_URL=http://localhost:9200 TF_ACC=TRUE go test -v -cover . -run TestAccElasticsearchIndex -count=1

all: $(NAME)