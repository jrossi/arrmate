
```shell
go build -o arrmate cmd/arrmate/main.go
```

```shell
./arrmate  config set discord.token=XXXXXXXXXXXXXXXXXXXXXXXXXXXX
./arrmate config set plex.url http://192.168.1.5:32400
./arrmate config set plex.token=XXXXXXXXXXXXXXXXX
./arrmate config set starr.sonarr.token=XXXXXXXXXXXXXXXXX
./arrmate config set starr.sonarr.url=http://192.168.1.5:8989/
./arrmate config list 
```


# run server 
```shell
./arrmate serve 
```
