# rw-cli

**WIP**

## Contributing

On MacOS, install Go by following the instructions here:
https://go.dev/doc/install
Don't use homebrew, it's not worth it.

We develop using VSCode with the official Go extension.

Just run `make` in your terminal and you'll get an `rw` binary in the root. Run
it by just doing `./rw`

When running `./rw` you'll get debug logs printed to `~/.rw/debug.log`.
`tail -f ~/.rw/debug.log` might be useful to run in a separate terminal window
in VSCode.
