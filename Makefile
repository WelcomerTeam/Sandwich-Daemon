all:
	go build -v -o ./out/sandwich
web:
	cd sandwich && npm i --force && npm run build
