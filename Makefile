.PHONY: assets cli

default: cli

assets: abiplay-assets authuiui-assets

abiplay-assets:
	@# (cd ui && yarn build)
	mkdir -p assets/abiplay
	go-bindata -o assets/abiplay/assets.go -pkg abiplay -prefix abiplay/build abiplay/build

authuiui-assets:
	@# (cd ui && yarn build)
	mkdir -p assets/authui
	go-bindata -o assets/authui/assets.go -pkg authui -prefix ui/build ui/build

cli:
	GOOS=darwin go build -o build/recryptportal-mac github.com/hayeah/recrypt-portal/cli/recryptportal
	GOOS=linux go build -o build/recryptportal-linux github.com/hayeah/recrypt-portal/cli/recryptportal
