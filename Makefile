
NUM_LINKED_GOMODS=`cat go.mod | grep -v "^\/\/" | grep replace | wc -l | sed -e "s/ *//g"`

all: build run

run:
	cp .env /tmp 
	cp .env.dev /tmp
	VIBERUNNER_ENV=dev VIBERUNNER_WEB_PORT=:8085 air

checklinks:
	@if [ x"${NUM_LINKED_GOMODS}" != "x0" ]; then	\
		echo "You are trying to deploying with symlinks.  Remove them first and make sure versions exist" && false ;	\
	fi

build: webbuild resymlink

webbuild:
	npm run build

resymlink:
	mkdir -p locallinks
	rm -Rf locallinks/*
	cd locallinks && ln -s ../../templar
	cd locallinks && ln -s ../../goutils
	cd locallinks && ln -s ../../oneauth
	cd locallinks && ln -s ../../s3gen
