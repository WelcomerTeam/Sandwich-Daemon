echo "Build Web Distributable"

cd web
npm run build
cd ..

echo "Docker build and push"
docker build --tag ghcr.io/welcomerteam/sandwich-daemon:next .
docker push ghcr.io/welcomerteam/sandwich-daemon:next
