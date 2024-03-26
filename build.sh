echo "Build Web Distributable"

cd web
npm run build
cd ..

echo "Simplify"
gofmt -l -s -w .

echo "Docker build and push"
docker build --tag sandwich-daemon:latest .
docker push ghcr.io/welcomerteam/sandwich-daemon:latest
