
.PHONY: build
build:
	docker-compose build

.PHONY: start
start:
	sudo docker-compose up

.PHONY: stop
stop:
	docker-compose stop

.PHONY: clean
clean:
	sudo rm -rf .artifacts

.PHONY: prune
prune:
	docker system prune

.PHONY: restart
restart:
	make build
	make start