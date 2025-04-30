build:
	docker build -f Dockerfile -t starql .
build-pg:
	docker build -f minidev_pgsql.Dockerfile -t starql-db .