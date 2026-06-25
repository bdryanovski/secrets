# Secrets

This is a secret manager design to to store and share across application and machine secrets.
Similar to what bitwarden, lastpass and other are doing but into more local and private way.

It should integrate with other services and fetch and share data to them.

### Features

- Should be able to integrate with the shell and provide env variables safely. So for example
  storing service tokens and keys - and consumming them with shell env vairable.
- Should be able to keep this data encrypted into sqlite database - and no one should be able to
  unlock the database without providing the encryption password or ssh key
- Should be able to store password for websites and logins, by showing me the username, sending the
  password into my clipboard and after 1min removing it from the clipboard - making sure that this
  1min is configurable
- Should be able to sync between machines - so first I need to register the machine so every
  instance should have uniq fingerprint used to encrypt the database, and generating a specific key
  - that key will be used to encrypt the state of the database - send to the other machine unpack
    using the key end encrypt it back locally - this way the data is on multi step secured. This
    could be unclocked only for machine that the export is meant to.
- There should be network interface for doing this sync so the machine will open a port with
  specific address and all this will be done between them. Making sure that all conflicts are
  resolved.
- There should be a way to generate random password for given email
- env keys should have there own different set of env for example staging, production, development
  and so on. So one key stored into the system could have multiple different version depending on
  the ask

This tool should run as a cli tool - should have very basic UI for seeing the env variables, and
account logins, adding new one, removing them, viewing the actual password, do basic CRUD operations
searching for it.

We need to have the ability to import data from Bitwarden, Apple Password manager, and other by
consuming imports or something else. and link it to the external storage.

Build it with Golang and use Bubbletea as CLI TUI - build everyting into modules and make sure the
are incapsolated and easy to understand.

Create a single Makefile to organize the project - build, clean and run - make sure to have build
for ios and build for linux - both platform i'm interested to have an build and rady to run.

Create me github actioan that will build a version and package every time when new PR are merged
into main/master branch - keep the version somewhere and modify it easily - because incrementing
numbers in action is hard make the version number of the package like that -
<year>.<month>.<rc>-rc<hash> and hash in this context is commit hash - rc should be number that
start from 0 and go with one number up every time when we merge something until end of time

database should be encrypted not inspectable from outside should be located into
~/.config/secrets/database.db
