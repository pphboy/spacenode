.PHONY: all linux-client app-client win-client lzcspacenode server ui

all: dist-dirs linux-client app-client win-client lzcspacenode server ui bd-lpk

dist-dirs:
	rm -rf dist
	mkdir -p dist/client dist/ui

linux-client: dist-dirs
	cd backend/node/linux && CGO_ENABLED=1 CC=musl-gcc GOOS=linux GOARCH=amd64 go build -ldflags "-X main.BuildNodeType=client" -o ../../../dist/client/spacenode-client-linux

win-client: dist-dirs
	cd backend/node/win && GOOS=windows GOARCH=amd64 go build -ldflags "-X main.BuildNodeType=client" -o ../../../dist/client/spacenode-client-win.exe

lzcspacenode: dist-dirs
	#cd backend/node/linux && CGO_ENABLED=1 CC=musl-gcc GOOS=linux GOARCH=amd64 go build -ldflags "-X main.BuildNodeType=app" -o ../../../dist/client/lzcspacenode
	cd backend/node/linux && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.BuildNodeType=app" -o ../../../dist/client/lzcspacenode

server: dist-dirs
	cd backend && CGO_ENABLED=1 CC=musl-gcc GOOS=linux GOARCH=amd64 go build -ldflags "-X main.BuildNodeType=client" -o ../dist/server

ui: dist-dirs
	cd ui && npm run build && cp -r dist/* ../dist/ui/

bd-lpk: dist-dirs
	lzc-cli project build -f lzc-build.release.yml

