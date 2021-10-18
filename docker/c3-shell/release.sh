export VER=1.52

docker build -t c3ai/c3-impala-shell:$VER .
docker tag c3ai/c3-impala-shell:$VER c3ai/c3-impala-shell:latest
docker push c3ai/c3-impala-shell:$VER
docker push c3ai/c3-impala-shell:latest
