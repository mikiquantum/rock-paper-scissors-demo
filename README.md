# rock-paper-scissors-demo
Go Demo Rock-Paper-Scissors with libp2p

This is the code for a quick introduction on how to use libp2p at the Go developers meetup in Berlin. Slides can be found here:
https://docs.google.com/presentation/d/1q6oE7xa1EyrlG5WimlvTnOKoCOHE3071L3QsvHzldjs

**Install Dep**

`curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh`

**Install Dependencies**

`dep ensure`

**Build**

`go build -o player player.go`

**Run - 3 different terminals**

*Run DHT Bootstrap Node*

`./player 30000 dht bootstrap`

*Run Player 1 Node*

`./player 30001 enc_1`

*Run Player 2 Node*

`./player 30002 enc_2`
