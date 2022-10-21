


1 - Use `make` to build the program you want. That will leave an executable in the `bin` directory inside the repository.

```sh
$ make osxkeychain
```

2 - Put that binary in your `$PATH`, so Docker can find it.

```sh
$ sudo cp bin/build/docker-credential-osxkeychain /usr/local/bin/docker-credential-intelosxkeychain
```

3 - Start docker registry instance
```sh
docker compose up
```

4 - Edit ~/.docker/config.json to use intelosxkeychain.
```json
{
  "credsStore": "intelosxkeychain"
}
```

5
```
docker image tag hello-world localhost:5043/myfirstimage
docker push localhost:5043/myfirstimage
```

