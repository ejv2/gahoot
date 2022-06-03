# Gahoot - Distributed, FOSS rewrite of Kahoot! in GO
# Copyright 2022 - Ethan Marshall

SRV_SRC = main.go front.go ver.go \
	  config/conf.go config/parse.go
EXE     = gahoot

TSC_SRC = frontend/src/index.ts
TSC_OUT = frontend/static/js/index.js
TSC_DEP = frontend/node_modules

all: server frontend

server: ${EXE}

frontend: ${TSC_OUT}

${EXE}: ${SRV_SRC}
	go build .

${TSC_OUT}: ${TSC_SRC} ${TSC_DEP}
	cd frontend && npm run build

${TSC_DEP}:
	cd frontend && npm install

watch:
	find -name "*.go" -print -or -name "*.ts" -print -or -name "node_modules" -prune \
		| entr -cs "pkill gahoot; make; ./gahoot &"; pkill gahoot

clean:
	rm -f ${EXE}
	rm -rf frontend/static/js/

.PHONY: clean watch frontend server
