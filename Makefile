# Gahoot - Distributed, FOSS rewrite of Kahoot! in GO
# Copyright 2022 - Ethan Marshall

SRV_SRC = main.go front.go play.go api.go ver.go \
	  config/conf.go config/parse.go \
	  game/game.go game/doc.go game/coordinator.go game/client.go game/host.go game/player.go game/action.go \
	  game/quiz/quiz.go game/quiz/manager.go
EXE     = gahoot

TSC_SRC = frontend/src/index.ts frontend/src/play.ts frontend/src/host.ts frontend/src/find.ts
TSC_OUT = frontend/static/js/index.js frontend/static/js/play.js
TSC_DEP = frontend/node_modules

all: server frontend

server: ${EXE}

frontend: ${TSC_OUT}

test:
	go test ./...

${EXE}: ${SRV_SRC}
	go build .

${TSC_OUT}: ${TSC_SRC} ${TSC_DEP}
	cd frontend && npm run build

${TSC_DEP}:
	cd frontend && npm install

watch:
	find -name "*.go" -print -or -name "*.ts" -print -or -name "node_modules" -prune \
		| entr -cs "pkill gahoot; make; ./gahoot &"; pkill gahoot

frontwatch:
	cd frontend && npm run watch

clean:
	rm -f ${EXE}
	rm -rf frontend/static/js/
	rm -rf frontend/.genjs/

.PHONY: clean test watch frontend server
