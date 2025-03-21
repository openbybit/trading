#!/bin/bash
file=.pid

start() {
    if [[ -e $file ]]; then
        echo "server already started, pid: `cat ${file}`"
        return
    else
        echo "start server as daemon"
        ./bgw serve http -d 
        echo "server started"
    fi
}

stop () {
    if [[ ! -e $file ]]; then
        return
    else
        echo "stop server"
        kill -9 $(cat $file) > /dev/null
        rm -f $file
        echo "server stopped"
    fi
}

case $1 in
stop)
    stop
;;
start)
    start
;;
*)
    stop
    start
;;
esac
