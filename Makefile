dev:
	make create_aof
	docker-compose up --build

create_aof:
	chmod +x create_aof.sh
	./create_aof.sh
