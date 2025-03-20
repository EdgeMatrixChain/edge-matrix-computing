# P2P computing network with EdgeMatrixComputing

You can create a P2P network as shown below and connect the webapp to the edge of the network:
```
+----------------+                 +----------------+                 +-----------------+            +-----------------+
|                |                 |                |                 |                 |            |                 |
|                | libp2p stream   |                | libp2p stream   |                 |  HTTP      |                 |
|   Relay Node   <----------------->   Relay Node   <----------------->    Edge Node    <------------>   Webapp SERVER |
|                |   Discovery     |                | p2p Reservation |                 |            |                 |
|  libp2p host   |                 |  libp2p host   |                 |   libp2p host   |            |   local host    |
+------|---------+                 +-------|--------+                 +-----------------+            +-----------------+                   
   libp2p stream                     libp2p stream 
       |                                   |  
   Discovery                           Discovery
+------|---------+                 +-------|--------+                 +-----------------+            +-----------------+
|                |                 |                |                 |                 |            |                 |
|                | libp2p stream   |                | libp2p stream   |                 |  HTTP      |                 |
|   Relay Node   <----------------->   Relay Node   <----------------->    Edge Node    <------------>   Webapp SERVER |
|                |   Discovery     |                | p2p Reservation |                 |            |                 |
|  libp2p host   |                 |  libp2p host   |                 |   libp2p host   |            |   local host    |
+------|---------+                 +-------|--------+                 +-----------------+            +-----------------+                   
   libp2p stream                     libp2p stream 
       |                                   |  
   Discovery                           Discovery
+------|---------+                 +-------|--------+                 +-----------------+            +-----------------+
|                |                 |                |                 |                 |            |                 |
|                | libp2p stream   |                | libp2p stream   |                 |  HTTP      |                 |
|   Relay Node   <----------------->   Relay Node   <----------------->    Edge Node    <------------>   Webapp SERVER |
|                |   Discovery     |                | p2p Reservation |                 |            |                 |
|  libp2p host   |                 |  libp2p host   |                 |   libp2p host   |            |   local host    |
+------|---------+                 +-------|--------+                 +-----------------+            +-----------------+                   
   libp2p stream                     libp2p stream 
       |                                   |  
      ...                                 ...
```

The following example shows how to create a P2P network and how to call a webapp connected to edge nodes：

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
make get
make build
make build-example
cd build
```

## Usage
#### Step1: Initialize keys for relay nodes

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

#### Step 2: Generate an example. json file
It will print a local peer address. If you would like to run this on a separate machine, please replace the IP accordingly,
And replace the nodeId with the actual text from the Step1.
```sh
 ./edge-matrix-computing genesis --dir example.json --name MyNetwork --network-id 1000 --bootnode=/ip4/127.0.0.1/tcp/50001/p2p/16Uiu2HAmKdcZsHngqMFzXjrhzWHBPJTDFjj9sdeEYVHqA1nm4hQr --bootnode=/ip4/127.0.0.1/tcp/51001/p2p/16Uiu2HAm7U1QtzHESv44Pvg6eGkA6cr9pewVauiYRPkfDqoD2SQd
```

#### Step 3: Start relay nodes
```sh
./edge-matrix-computing server --network example.json --data-dir node_1  --grpc-address 0.0.0.0:50000 --libp2p 0.0.0.0:50001 --jsonrpc 0.0.0.0:50002 --relay-libp2p 0.0.0.0:50004 --trans-proxy 0.0.0.0:50005 --relay-discovery  --app-no-auth --app-no-agent
./edge-matrix-computing server --network example.json --data-dir node_2  --grpc-address 0.0.0.0:51000 --libp2p 0.0.0.0:51001 --jsonrpc 0.0.0.0:51002 --relay-libp2p 0.0.0.0:51004 --trans-proxy 0.0.0.0:51005 --relay-discovery  --app-no-auth --app-no-agent
```

#### Step 4: Generate an example_edge. json file
Replace the nodeId with the actual text from the Step1.
```sh
 ./edge-matrix-computing genesis --dir example_edge.json --name MyNetwork --network-id 1000 --relaynode=/ip4/127.0.0.1/tcp/50004/p2p/16Uiu2HAmKdcZsHngqMFzXjrhzWHBPJTDFjj9sdeEYVHqA1nm4hQr --relaynode=/ip4/127.0.0.1/tcp/51004/p2p/16Uiu2HAm7U1QtzHESv44Pvg6eGkA6cr9pewVauiYRPkfDqoD2SQd
```

#### Step 5: Initialize the key for the edge node1

```sh
./edge-matrix-computing secrets init --data-dir edge_1 
```
Output:
```
[SECRETS INIT]
Public key (address) = 0x69Ea3778e328B0De0E61aE3941d48a75DB50cB6F
Node ID              = 16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie
```

#### Step 6: Start the edge node
After 25 seconds of startup, the information of edge nodes will be synchronized to the P2P network
```sh
./edge-matrix-computing server --network example_edge.json --data-dir edge_1  --grpc-address 0.0.0.0:52000  --libp2p 0.0.0.0:52001 - --relay-on --running-mode edge --app-url http://127.0.0.1 --app-no-auth --app-no-agent
```

#### Step 7: Start the webapp
```sh
./webapp
```

As you can see, the prints the listening address `localhost:9527`.

You can now use this webapp through P2P networks, for example with `curl`. Please replace the nodeId with the actual text from the Step5.

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

## Subscribe node event
Edge nodes will broadcast their status to the network every 15 minutes or when they reconnect after a relay interruption. Establish a Websocket connection to the jsonRPC port of any relay node and send a subscription request to receive events from the edge node. The event information includes the basic information of the edge node, including the connected relay address and relay proxy service port.

Example of jsonRPC address
````
ws://127.0.0.1:50002/edge_ws
````

Example of all nodes request:
````
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "edge_subscribe",
  "params": [
    "node",
    { "name":"", "id": "", "version":""}
  ]
}
````
Example of nodeId=16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie request:
````
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "edge_subscribe",
  "params": [
    "node",
    { "name":"", "id": "16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie", "version":""}
  ]
}
````

Received notifications
````
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "aad6f294-5c4d-4a64-b480-b97122c75d5e"
}

{
    "jsonrpc": "2.0",
    "method": "edge_subscription",
    "params": {
        "subscription": "aad6f294-5c4d-4a64-b480-b97122c75d5e",
        "result": {
            "Name": "",
            "Tag": "",
            "Version": " Build",
            "PeerID": "16Uiu2HAkzBCWtZq49xzn4HcsGw7NZHSuSSS97HfzLyMDyY9KTDie",
            "IpAddr": "127.0.*.*",
            "AppOrigin": "{\"appOrigin\":\"edgematrix:LLM-ChatBot,edgematrix:deepseek7b\",\"gpuInfo\":\"[{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"},{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"},{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"},{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"},{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"},{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"},{\\\"gpuMemory\\\":\\\"24\\\",\\\"gpuModel\\\":\\\"NVIDIA GeForce RTX 3090\\\"}]\"}",
            "ModelHash": "",
            "Mac": "0c:42:a1:1f:33:2e",
            "MemInfo": "{\"total\": 1082048880640, \"free\":845533614080, \"used_percent\":0.761393}",
            "CpuInfo": "{\"Cpus\":64,\"VendorId\":\"AuthenticAMD\",\"Family\":\"23\",\"Model\":\"49\",\"Cores\":1,\"ModelName\":\"AMD EPYC 7302 16-Core Processor\",\"Mhz\":3000}",
            "GpuInfo": "{\"gpus\":8,\"graphics_card\":[\"card #0  [affined to NUMA node 0]@0000:63:00.0 -\\u003e driver: 'ast' class: 'Display controller' vendor: 'ASPEED Technology, Inc.' product: 'ASPEED Graphics Family'\",\"card #1  [affined to NUMA node 0]@0000:61:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\",\"card #2  [affined to NUMA node 0]@0000:41:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\",\"card #3  [affined to NUMA node 0]@0000:01:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\",\"card #4  [affined to NUMA node 1]@0000:e1:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\",\"card #5  [affined to NUMA node 1]@0000:c1:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\",\"card #6  [affined to NUMA node 1]@0000:81:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\",\"card #7  [affined to NUMA node 1]@0000:a1:00.0 -\\u003e driver: 'nvidia' class: 'Display controller' vendor: 'NVIDIA Corporation' product: 'GA102 [GeForce RTX 3090]'\"]}",
            "StartupTime": 1739869483739,
            "Uptime": 94485191,
            "GuageHeight": 0,
            "GuageMax": 0,
            "AveragePower": 0,
            "RelayHost": "127.0.0.1",
            "RelayProxyPort": 50005
        }
    }
}
...
````
## License
MIT License
