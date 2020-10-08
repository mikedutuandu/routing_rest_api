
init:
	mkdir -p assets/orders
	mkdir -p logs/

build:
	go mod download
	@[ -f ./janio-backend ] && cp janio-backend janio-backend_bk || true
	env GOOS=linux GOARCH=amd64 GOARM=7 go build -o janio-backend server.go

reload:
	sudo supervisorctl reload

status:
	sudo supervisorctl status

rollback:
	@[ -f ./janio-backend_bk ] && mv ./janio-backend_bk ./janio-backend || true

clean:
	@[ -f ./janio-backend ] && rm -rf ./janio-backend || true
	@[ -f ./janio-backend_bk ] && rm -rf ./janio-backend_bk || true

start:
	~/go/bin/CompileDaemon -build="go build -o janio-backend ./server.go" -command="./janio-backend"

# alias janio.dev='ssh -i /home/vietnguyen/keys/janio_dev.pem ubuntu@ec2-3-0-95-18.ap-southeast-1.compute.amazonaws.com'

#ssh root@68.183.230.245
#minhthuaA@1102
#cd /var/www/janio-backend
#tail -f nohup.out