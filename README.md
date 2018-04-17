# rock-paper-scissors-demo
Go Demo Rock-Paper-Scissors with libp2p

**Install Glide**

`curl https://glide.sh/get | sh`

**Install Dependencies**

`glide install`

**Build**

`go build -o player player.go`

**Run - 3 different terminals**

*Run DHT Bootstrap Node*

`./player 30000 dht bootstrap`

*Run Player 1 Node*

`./player 30001 enc_1`

*Run Player 2 Node*

`./player 30002 enc_2`