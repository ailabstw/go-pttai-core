---
title: 'Example'
date: 2019-02-11T19:27:37+10:00
weight: 2
---

Here's a basic example of running a single node. You can check out the full source [here](https://github.com/ailabstw/go-pttai-core/tree/master/examples/pttai).

### Run the example

Build the example from source:

```
git clone git@github.com:ailabstw/go-pttai-core.git
cd examples/basic
go build .
```

Run the example:

```
./basic ./tmp 14779
```

The command above will start a node at port `14779` and save its data at `./tmp`.

### Interact with node

Now we have a node running. How can we interact with it?

By default, `PTT.ai-core` provides a json rpc API server. We can communicate with it with `curl`:

```
$ curl --header "Content-Type: application/json" \
    --request POST \
    --data '{"id": "testCall", "method": "me_get", "params": []}' \
    http://localhost:14779
```

Here's a sample response:

```
{
    "id": "testCall",
    "jsonrpc": "2.0",
    "result": {
        "CT": {
            "NT": 458761000,
            "T": 1556780390
        },
        "ID": "BXqNU9FBdxA7KKpAvm4fpEn6iCFzd8cZ8cSoDzUNfaj5hkbLUFByBia",
        "NodeID": "ab1852660163659efa14c29135cb95dbf4c4875d534694fbbbf6cd97f7cda4535bff8e72c9885c497036ef3c81f0212e340b149388def5e9c69590900958d9c4",
        "RaftID": 2873960715719249470,
        "S": 7,
        "UT": {
            "NT": 458761000,
            "T": 1556780390
        },
        "V": 2
    }
}
```
