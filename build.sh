echo "Build GO Executable"
go build -v -o sandwich cmd/main.go

echo "Build Web Distributable"
#!cd web
#!yarn lint
#!yarn build --modern
#!cd ..

echo "Docker build and push"
docker build --tag 1345/sandwich-daemon:latest .
docker push 1345/sandwich-daemon:latest
