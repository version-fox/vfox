# Sample plugin

From the vfox repository root:

```sh
go run . plugin test examples/plugins/sample
go run . plugin run examples/plugins/sample PreInstall --input '{"version":"1.2.3"}' --os linux --arch amd64 --offline --json
```

The tests use fixed HTTP fixtures and exercise real hooks, JSON parsing, version selection, error recovery, and platform/environment overrides. They do not download an SDK. The example URLs are placeholders.

See [the author guide](../../../docs/plugins/create/testing.md) ([中文](../../../docs/zh-hans/plugins/create/testing.md)).
