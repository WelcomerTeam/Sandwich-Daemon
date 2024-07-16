all:
	go build -v -o ./out/sandwich
web:
	cd web && npm i --force && npm run build
