# P2P computing network with EdgeMatrixComputing

This example shows how to create a P2P computing network with EdgeMatrixComputing:

```
                  +----------------+                +-----------------+              +-----------------+
 HTTP Request     |                |                |                 |              |                 |
+----------------->                | libp2p stream  |                 |  HTTP        |                 |
                  |   Relay Node   <---------------->    Edge Node    <------------->    Webapp SERVER |
<-----------------+                |                |                 | Req & Resp   |                 |
  HTTP Response   |  libp2p host   |                |   libp2p host   |              |   local host    |
                  +----------------+                +-----------------+              +-----------------+                   
```

## Build

From the `edge-matrix-computing` directory run the following:

```
make import-core
make build
make build-example
cd build
```

## Usage
Step1: Initialize keys for network nodes:

Initialize the key for the relay node1
```sh
./edge-matrix-computing secrets init --data-dir node_1  
```
Output:
```
[SECRETS INIT]
Public key (address) = 0xcEd3Ecad2dC824f3aFb1559529Abf9ADC5F63E36
Node ID              = 16Uiu2HAmKdcZsHngqMFzXjrhzWHBPJTDFjj9sdeEYVHqA1nm4hQr
```

Initialize the key for the relay node2
```sh
./edge-matrix-computing secrets init --data-dir node_2  
```
Output:
```
[SECRETS INIT]
Public key (address) = 0x6D6Ca00263922bcFa38B7884AA9a2C97069B4454
Node ID              = 16Uiu2HAm7U1QtzHESv44Pvg6eGkA6cr9pewVauiYRPkfDqoD2SQd
```

Initialize the key for the edge node1

```sh
./edge-matrix-computing secrets init --data-dir edge_1 
```
Output:
```
[SECRETS INIT]
Public key (address) = 0x69Ea3778e328B0De0E61aE3941d48a75DB50cB6F
Node ID              = 16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie
```

Step 2: Generate an example. json file, It will print a local peer address. If you would like to run this on a separate machine, please replace the IP accordingly:
```sh
 ./edge-matrix-computing genesis --dir example.json --name MyNetwork --network-id 1000 --bootnode=/ip4/127.0.0.1/tcp/50001/p2p/16Uiu2HAmKdcZsHngqMFzXjrhzWHBPJTDFjj9sdeEYVHqA1nm4hQr --bootnode=/ip4/127.0.0.1/tcp/51001/p2p/16Uiu2HAm7U1QtzHESv44Pvg6eGkA6cr9pewVauiYRPkfDqoD2SQd
```

Step 3: Start relay nodes
```sh
./edge-matrix-computing server --network example.json --data-dir node_1  --grpc-address 0.0.0.0:50000 --libp2p 0.0.0.0:50001 --jsonrpc 0.0.0.0:50002 --relay-libp2p 0.0.0.0:50004 --trans-proxy 0.0.0.0:50005 --relay-discovery --app-no-agent
./edge-matrix-computing server --network example.json --data-dir node_2  --grpc-address 0.0.0.0:51000 --libp2p 0.0.0.0:51001 --jsonrpc 0.0.0.0:51002 --relay-libp2p 0.0.0.0:51004 --trans-proxy 0.0.0.0:51005 --relay-discovery --app-no-agent
```

Step 4: Generate an example_edge. json file
```sh
 ./edge-matrix-computing genesis --dir example_edge.json --name MyNetwork --network-id 1000 --relaynode=/ip4/127.0.0.1/tcp/50004/p2p/16Uiu2HAmKdcZsHngqMFzXjrhzWHBPJTDFjj9sdeEYVHqA1nm4hQr --relaynode=/ip4/127.0.0.1/tcp/51004/p2p/16Uiu2HAm7U1QtzHESv44Pvg6eGkA6cr9pewVauiYRPkfDqoD2SQd
```

Step 5: Start the edge node， After 25 seconds of startup, the information of edge nodes will be synchronized to the P2P network
```sh
./edge-matrix-computing server --network example_edge.json --data-dir edge_1  --grpc-address 0.0.0.0:52000  --libp2p 0.0.0.0:52001 - --relay-on --running-mode edge --app-url http://127.0.0.1 --app-no-auth --app-no-agent
```

Step 6: Start the webapp
```sh
./webapp
```

As you can see, the prints the listening address `localhost:9527`.

You can now use this webapp through P2P networks, for example with `curl`:

```
curl --location 'http://127.0.0.1:50005/edge/16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie/9527/echo' \
--header 'Content-Type: application/json' \
--data '{"message":"hello"}'

```
Or use another relay node
```
curl --location 'http://127.0.0.1:51005/edge/16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie/9527/echo' \
--header 'Content-Type: application/json' \
--data '{"message":"hello"}'

```

Response: 
{
"message": "hello"
}

You can also open a browser and enter the address http://127.0.0.1:50005/edge/16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie/9527/home Then you can see the home page