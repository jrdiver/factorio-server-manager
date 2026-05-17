docker compose -f docker/docker-compose.release.yaml build --no-cache
docker compose -f docker/docker-compose.release.yaml push
