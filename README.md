FUSE Hackathon
===============

Implement a fuse filesystem for generic filtering tasks.

```

./test.sh

```

To Inject as a particular user, you can do this:
```
curl -X POST --data-binary @danicaclaims.json http://127.0.0.1:9494/jwt-for-pid/$$
```
