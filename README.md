![](nebula.png)

A Real-Time data and event stream platform

## Proto

To generate the proto

```
$ protoc -I ./proto ./proto/*.proto --go_out=plugins=grpc:internal/cluster
```

## Notes

Currently the node discovery method _is_ partial partition tolerant when there is at least one path to the partitioned nodes. However the broadcast mechanism is not. For now this means that events do not cross partitions.

Some ideas for addressing this could be to hold on to messages for suspicious [read: previously reachable] node, with some form of configurable drop off for when to expire that backlog.

Since this would only affect real-time notifications this will be accepable for now. This assumes that any persistant changes are stored elsewhere.
