# Instructions 

Quick test.
```shell
docker run -p 8080:8080 registry.hub.docker.com/im7mortal/release-analyzer:latest
curl http://localhost:8080/apache/airflow/bloat\?start=v2.8.3\&end=v2.9.2
```

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

# Notes

For now I implemented the simplest asset name matching which satisfies `apache(- or _)airflow-(version).tar.gz` pattern. 
As next step I would use regex for string matching.
```golang
	for _, asset := range release.Assets {
		if strings.HasSuffix(*asset.BrowserDownloadURL, ".tar.gz") &&
			!strings.Contains(*asset.BrowserDownloadURL, "source") {
			// ...
		}
	}
```
