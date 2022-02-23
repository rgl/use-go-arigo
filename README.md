Example on how to use the [arigo](https://github.com/myanimestream/arigo) library to download a ISO.

**NB this is not currently working due to [#2](https://github.com/myanimestream/arigo/issues/2)**

## Notes

* Although there is a notification WebSocket, [there is no `downloadProgress` event](https://github.com/aria2/aria2/issues/839).
    * Instead we have to poll the status using `GID.TellStatus`.

## References

* [tellStatus](https://aria2.github.io/manual/en/html/aria2c.html#aria2.tellStatus)
