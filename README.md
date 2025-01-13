# Instructions 

## Run

The fastest way to run it. Is use [Dockerfile](Dockerfile). 
```shell
docker build -t release-analyzer:latest . && docker run -p 8080:8080 release-analyzer:latest
```

Or run [run.sh](run.sh)

Alternatively, you can compile the Go binary for your operating system using the command:
```shell
go build .
```
