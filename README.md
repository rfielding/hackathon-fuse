FUSE Hackathon
===============

Implement a fuse filesystem for generic filtering tasks.

```

./test.sh '{"claims":{"values":{"company":["ford","decipher"],"email":["joe.fielding@gmail.com"]}}}'

./test.sh '{"claims":{"values":{"company":["mcd"],"email":["bob@gmail.com"]}}}'

./test.sh '{"claims":{"values":{"email":["danica.fielding@gmail.com"]}}}'

./test.sh '{"claims":{"values":{"email":["some@yahoo.com"]}}}'

```
