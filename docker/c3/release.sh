export VER=1.5

docker build -t c3ai/c3-impala-hms:$VER .
docker tag c3ai/c3-impala-hms:$VER c3ai/c3-impala-hms:latest
docker push c3ai/c3-impala-hms:$VER
docker push c3ai/c3-impala-hms:latest
