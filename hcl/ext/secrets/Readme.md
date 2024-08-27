Secret values for tests were generated using [`ecies` cli](https://github.com/chronicleprotocol/configs/pull/23). 

Here is the example invocation:

```
echo "hello world" | \
./ecies encrypt --keystore=$(find testdata/keystore -type f) --passphrase="helloworld" -o hex
```

