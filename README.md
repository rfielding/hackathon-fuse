FUSE Hackathon
===============

Implement a fuse filesystem for generic filtering tasks.

```

./test.sh '{"claims":{"values":{"company":["ford","decipher"],"email":["rob.fielding@gmail.com"]}}}' rob

./test.sh '{"claims":{"values":{"company":["mcd"],"email":["bob@gmail.com"]}}}' bob

./test.sh '{"claims":{"values":{"email":["danica.fielding@gmail.com"]}}}' danica

./test.sh '{"claims":{"values":{"email":["some@yahoo.com"]}}}' some

```
