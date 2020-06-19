# Running the Doomsday docker image

There are a few ways to run the doomsday image - the first of which is to specify no options other than the port mapping. This will spin up a doomsday server on localhost with a default config. Not exactly useful unless you want to monitor the example configs.

```bash
docker run -d -p 8111:8111 doomsdayproject/doomsday
```

There are two methods to specify your own config for the doomsday image to use, the first being mounting your config via a volume and setting the `DDAY_CONFIG_FILE` environment variable to the location of the config in the container.

For example, the below command mounts a `config` directory containing `ddayconfig.yml` into the container.

```bash
docker run -d -p 8111:8111 -v $(pwd)/config/:/doomsday/config -e DDAY_CONFIG_FILE=/doomsday/config/ddayconfig.yml doomsdayproject/doomsday
```

If you don't want to mount in the config via a file, you can also specify the entire config in an environment variable `DDAY_CONFIG`.

```bash
export DDAY_CONFIG=$(cat config/ddayconfig.yml)
docker run -d -p 8111:8111 -e DDAY_CONFIG="$DDAY_CONFIG" doomsdayproject/doomsday
```