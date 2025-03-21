build-press:
	go build -o bin/bgw_press ./main.go

start-press:
	nohup ./bin/bgw_press pressureTest -s=100 -u=20000 -C=1 --addr=10.111.19.140:8081 &

stop-press:
	pkill bgw_press

show:
	ps -ef | grep bgw

build_bgw:
	go build -o ./bin/bgw ./main.go