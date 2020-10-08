# Sample Echo with GORM



## Configurations
Configuration used gonfig. All configs are declared in `config/config.json`

## Architecture
| Folder | Details |
| --- | ---|
| api | Holds the api endpoints |
| db | Database Initializer and DB manager |
| route | router setup |
| model | Models|


## compile for ubuntu
env GOOS=linux GOARCH=amd64 GOARM=7 go build


