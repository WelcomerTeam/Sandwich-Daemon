echo "Build Web Distributable"
cd sandwich
yarn build
cd ..

echo "Simplify - gofmt"
gofmt -l -s -w .

echo "Simplify - gofumpt"
gofumpt -l -s -w .

echo "Simplify - gci"
gci -w .

echo "Simplify - goimports"
goimports -l -w .

echo "Docker build and push"
docker build --tag 1345/sandwich-daemon:latest .
docker push 1345/sandwich-daemon:latest
