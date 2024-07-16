echo "Build Web Distributable"
cd web
yarn build
cd ..

echo "Simplify"
gofmt -l -s -w . && gofumpt -l -w . && gci --write . && goimports -local -w .

echo "Docker build and push"
docker build --tag 1345/sandwich-daemon:latest .
docker push 1345/sandwich-daemon:latest
