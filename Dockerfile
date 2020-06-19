FROM golang:1.14 as build

ADD . /go/src/doomsdayproject/doomsday/
RUN cd /go/src/doomsdayproject/doomsday/ \
  && go get ./... \
  && make all

FROM ubuntu:18.04
EXPOSE 8111/tcp
COPY --from=build /go/src/doomsdayproject/doomsday/doomsday-linux /doomsday/doomsday
ADD ./docker/ /doomsday/
RUN adduser --system --disabled-password --no-create-home --home /doomsday/ doomsday \
  && chown -R doomsday /doomsday && chmod 755 /doomsday/entrypoint.sh
USER doomsday
CMD ["/doomsday/entrypoint.sh"]
