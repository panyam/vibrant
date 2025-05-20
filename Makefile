
vib := go run cmd/*.go
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

systemprompt:
	cat prompts/systemprompt.txt

prompt:
	source ~/personal/.shhelpers && files_for_llm `find . | grep -v apiclient | grep -v pnpm | grep -v llmprompts | grep -v node.mod | grep -v .git | grep -v gen | grep -v web.static | grep -v output | grep -v content | grep -v dist | grep -v ./static/css/tailwind.css | grep -v go.sum | grep -v pnpm | grep -v "\.png" | grep -v "\.jpeg" | grep -v "\.jpg" | grep -v LICENSE`

respond_to_createfile:
	$(vib) tools run -c create_file

respond_to_readfile:
	vib tools run -c read_file

tool_createfile:
	vib tools run create_file

tool_readfile:
	vib tools run read_file

respond_to_tool:
	vib tools respond
